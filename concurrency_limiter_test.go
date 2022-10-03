package circuitbreaker

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

func Test_concurrencyLimiter_acquireAndRelease(t *testing.T) {
	tests := []struct {
		name      string
		size      int64
		ampFactor int64
	}{
		{
			name:      "amplifying factor 1",
			size:      1,
			ampFactor: 1,
		},
		{
			name:      "amplifying factor > 1",
			size:      20,
			ampFactor: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := newConcurrencyLimiter(tt.size, 0)

			err := pool.acquire(context.Background(), tt.ampFactor)
			if err != nil {
				t.Errorf("acquire error = %v, want nil", err)
			}

			// should return error as resource is exhausted
			err2 := pool.acquire(context.Background(), tt.ampFactor)
			if err2 == nil {
				t.Errorf("acquire error, want non nil")
			}

			pool.release(context.Background(), tt.ampFactor)

			err3 := pool.acquire(context.Background(), tt.ampFactor)
			if err3 != nil {
				t.Errorf("acquire error = %v, want nil", err3)
			}
		})
	}
}

func Test_concurrencyLimiter_acquireAndRelease_zeroRequest(t *testing.T) {
	pool := newConcurrencyLimiter(1, 0)

	err := pool.acquire(context.Background(), 0)
	if err != nil {
		t.Errorf("acquire err = %v, want nil", err)
	}

	// Should still succeed because n = 0.
	err2 := pool.acquire(context.Background(), 0)
	if err2 != nil {
		t.Errorf("acquire err = %v, want nil", err2)
	}

	pool.release(context.Background(), 0)

	// Should be able to acquire ticket after return
	err3 := pool.acquire(context.Background(), 0)
	if err3 != nil {
		t.Errorf("acquire err = %v, want nil", err3)
	}
}

func Test_concurrencyLimiter_acquireAndRelease_blocking(t *testing.T) {
	tests := []struct {
		name      string
		size      int64
		ampFactor int64
	}{
		{
			name:      "amplifying factor 1",
			size:      1,
			ampFactor: 1,
		},
		{
			name:      "amplifying factor > 1",
			size:      20,
			ampFactor: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := newConcurrencyLimiter(tt.size, 10*time.Second)
			ctx := context.Background()

			err := pool.acquire(ctx, tt.ampFactor)
			if err != nil {
				t.Errorf("acquire error = %v, want nil", err)
			}

			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				// Second call will block up to maxWaitDuration.
				// Once release is called outside, then this will unblock.
				err2 := pool.acquire(ctx, tt.ampFactor)
				if err2 != nil {
					t.Errorf("acquire err = %v, want nil", err2)
				}
				wg.Done()
			}()

			pool.release(context.Background(), tt.ampFactor)
			wg.Wait()
			pool.release(context.Background(), tt.ampFactor)
		})
	}
}

func Test_concurrencyLimiter_acquireAndRelease_timeout(t *testing.T) {
	tests := []struct {
		name      string
		size      int64
		ampFactor int64
	}{
		{
			name:      "amplifying factor 1",
			size:      1,
			ampFactor: 1,
		},
		{
			name:      "amplifying factor > 1",
			size:      20,
			ampFactor: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := newConcurrencyLimiter(tt.size, 2*time.Second)
			ctx := context.Background()

			err := pool.acquire(ctx, tt.ampFactor)
			if err != nil {
				t.Errorf("acquire error = %v, want nil", err)
			}

			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				// Second call will block up to maxWaitDuration.
				// Once release is called outside, then this will unblock.
				err2 := pool.acquire(ctx, tt.ampFactor)
				if err2 == nil {
					t.Errorf("acquire err = %v, want nil", err2)
				}
				wg.Done()
			}()

			go func() {
				time.Sleep(4 * time.Second)
				pool.release(context.Background(), tt.ampFactor)
			}()

			wg.Wait()
			// pool.release(context.Background(), tt.ampFactor)
		})
	}
}

func TestGetCount(t *testing.T) {
	pool := newConcurrencyLimiter(3, 0)
	_ = pool.acquire(context.Background(), 1)
	_ = pool.acquire(context.Background(), 1)
	_ = pool.acquire(context.Background(), 1)
	_ = pool.acquire(context.Background(), 1)

	count := pool.getCount(true)
	if count != 3 {
		t.Errorf("getcount count = %v, want %v", count, 3)
	}

	uncappedCount := pool.getCount(false)
	if uncappedCount != 4 {
		t.Errorf("get uncapped count = %v, want %v", uncappedCount, 4)
	}
}

func AcquireReleaseByMutex(parallelism int, b *testing.B) {
	lock := sync.Mutex{}
	counter := 0

	b.SetParallelism(parallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// First lock to acquire permission
			lock.Lock()
			counter++
			lock.Unlock()

			// Second lock to release permission
			lock.Lock()
			counter--
			lock.Unlock()
		}
	})
}

func AcquireReleaseByChan(parallelism int, b *testing.B) {
	tickets := make(chan *struct{}, parallelism)
	for i := 0; i < parallelism; i++ {
		tickets <- &struct{}{}
	}

	b.SetParallelism(parallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Acquire permission
			ticket := <-tickets

			// Release permission
			tickets <- ticket
		}
	})
}

func BenchmarkAcquireRelease(b *testing.B) {
	maxThreads := 500000
	numCPU := runtime.NumCPU()
	for i := numCPU; i < maxThreads; i *= 2 {
		parallelism := i / numCPU
		b.Run(fmt.Sprintf("AcquireReleaseByMutex_%d", parallelism), func(b *testing.B) { AcquireReleaseByMutex(parallelism, b) })
		b.Run(fmt.Sprintf("AcquireReleaseByChan_%d", parallelism), func(b *testing.B) { AcquireReleaseByChan(parallelism, b) })
	}
}
