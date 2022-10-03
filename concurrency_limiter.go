package circuitbreaker

import (
	"container/list"
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type concurrencyLimiter interface {
	acquire(ctx context.Context, n int64) error
	release(ctx context.Context, n int64)
	getCount(isCapped bool) int
}

type noOpLimiter struct{}

func (noOpLimiter) acquire(ctx context.Context, n int64) error { return nil }
func (noOpLimiter) release(ctx context.Context, n int64) {
	// no-op
}
func (noOpLimiter) getCount(isCapped bool) int { return 0 }

type waiter struct {
	n     int64
	ready chan<- struct{} // Closed after semaphore acquired
}

func newConcurrencyLimiter(size int64, maxWaitDuration time.Duration) concurrencyLimiter {
	if size <= 0 {
		return &noOpLimiter{}
	}

	return &weightedSemaphore{
		size:            size,
		maxWaitDuration: maxWaitDuration,
	}
}

type weightedSemaphore struct {
	size                     int64
	cur                      int64
	maxWaitDuration          time.Duration
	uncappedConcurrencyCount int64
	mu                       sync.Mutex
	waiters                  list.List
}

// Acquire acquires the semaphore with a weight of n, blocking until resources are available OR ctx is done.
// On success, it returns nil.
// On failure, it returns ctx.Err() and leaves the semaphore unchanged.
//
// If ctx is already done, Acquire may still succeed without blocking.
func (w *weightedSemaphore) acquire(ctx context.Context, n int64) error {
	atomic.AddInt64(&w.uncappedConcurrencyCount, n)

	// 1. Resource enough
	w.mu.Lock()
	if w.size-w.cur >= n && w.waiters.Len() == 0 {
		w.cur += n
		w.mu.Unlock()
		return nil
	}

	if n > w.size {
		// Don't make other Acquire calls block on one that's doomed to fail.
		w.mu.Unlock()
		return fmt.Errorf("max_concurrency_reached")
	}

	// 2. Resource not enough and not wait
	if w.maxWaitDuration <= 0 {
		w.mu.Unlock()
		return fmt.Errorf("max_concurrency_reached")
	}

	// 3. Resource not enough and wait
	ctx, cancel := context.WithTimeout(ctx, w.maxWaitDuration)

	ready := make(chan struct{})
	wt := waiter{n: n, ready: ready}
	elem := w.waiters.PushBack(wt)
	w.mu.Unlock()

	defer cancel()
	select {
	case <-ctx.Done():
		err := fmt.Errorf("concurrency_wait_timeout")
		w.mu.Lock()
		select {
		case <-ready:
			// Acquired the semaphore after we were canceled. Rather than trying to
			// fix up the queue, just pretend we didn't notice the cancellation.
			err = nil
		default:
			isFront := w.waiters.Front() == elem
			w.waiters.Remove(elem)
			// If we're at the front and there're extra tokens left, notify other waiters.
			if isFront && w.size > w.cur {
				w.notifyWaiters()
			}
		}
		w.mu.Unlock()
		return err
	case <-ready:
		return nil
	}
}

func (w *weightedSemaphore) release(ctx context.Context, n int64) {
	atomic.AddInt64(&w.uncappedConcurrencyCount, -n)

	w.mu.Lock()
	w.cur -= n
	if w.cur < 0 {
		// Should not be possible unless there's an implementation bug, we'll log just in case.
		log.Printf("concurrency_limiter_released_more_than_held, semaphore:%v", w)
		w.cur = 0
		w.notifyWaiters()
		w.mu.Unlock()
		return
	}

	w.notifyWaiters()
	w.mu.Unlock()
}

func (w *weightedSemaphore) getCount(isCapped bool) int {
	var count int64
	if isCapped {
		w.mu.Lock()
		count = w.cur
		w.mu.Unlock()
	} else {
		count = atomic.LoadInt64(&w.uncappedConcurrencyCount)
	}

	return int(count)
}

func (w *weightedSemaphore) notifyWaiters() {
	for {
		next := w.waiters.Front()
		// 1. not more waiters blocked
		if next == nil {
			break
		}

		wt := next.Value.(waiter)
		// 2. not enough resource for this waiter
		if w.size-w.cur < wt.n {
			// Not enough tokens for the next waiter.
			// We could keep going (to try to find a waiter with a smaller request),
			// but under load that could cause starvation for large requests;
			// instead, we leave all remaining waiters blocked.
			//
			// Consider a semaphore used as a read-write lock, with N tokens, N readers, and one writer.
			// Each reader can Acquire(1) to obtain a read-lock.
			// The writer can Acquire(N) to obtain a write-lock, excluding all the readers.
			// If we allow the readers to jump ahead in the queue, the writer will starve —
			// there is always one token available for every reader.
			break
		}

		w.cur += wt.n
		w.waiters.Remove(next)
		close(wt.ready)
	}
}
