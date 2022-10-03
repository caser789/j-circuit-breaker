package circuitbreaker

import (
	"reflect"
	"testing"
)

func Test_countWindowMetric(t *testing.T) {
	w := newCountWindowMetric(5)

	snapshotGot := w.getSnapshot()
	snapshotWant := newSnapshot(&counter{})
	if !reflect.DeepEqual(snapshotGot, snapshotWant) {
		t.Errorf(" snapshot not equal, want=%v, got=%v", snapshotWant, snapshotGot)
	}

	w.update(successOutcome)
	snapshotGot = w.getSnapshot()
	snapshotWant = newSnapshot(&counter{TotalCount: 1})
	if !reflect.DeepEqual(snapshotGot, snapshotWant) {
		t.Errorf(" snapshot not equal, want=%v, got=%v", snapshotWant, snapshotGot)
	}
	arrayGot := w.array
	arrayWant := []*counter{
		{},
		{TotalCount: 1},
		{},
		{},
		{},
	}
	if !reflect.DeepEqual(arrayGot, arrayWant) {
		t.Errorf(" array not equal, want=%v, got=%v", arrayGot, arrayWant)
	}

	w.update(errorOutcome)
	snapshotGot = w.getSnapshot()
	snapshotWant = newSnapshot(&counter{TotalCount: 2, ErrorCount: 1})
	if !reflect.DeepEqual(snapshotGot, snapshotWant) {
		t.Errorf(" snapshot not equal, want=%v, got=%v", snapshotWant, snapshotGot)
	}
	arrayGot = w.array
	arrayWant = []*counter{
		{},
		{TotalCount: 1},
		{TotalCount: 1, ErrorCount: 1},
		{},
		{},
	}
	if !reflect.DeepEqual(arrayGot, arrayWant) {
		t.Errorf(" array not equal, want=%v, got=%v", arrayGot, arrayWant)
	}

	w.update(timeoutSuccessOutcome)
	snapshotGot = w.getSnapshot()
	snapshotWant = newSnapshot(&counter{TotalCount: 3, ErrorCount: 1, TimeoutCount: 1})
	if !reflect.DeepEqual(snapshotGot, snapshotWant) {
		t.Errorf(" snapshot not equal, want=%v, got=%v", snapshotWant, snapshotGot)
	}
	arrayGot = w.array
	arrayWant = []*counter{
		{},
		{TotalCount: 1},
		{TotalCount: 1, ErrorCount: 1},
		{TotalCount: 1, TimeoutCount: 1},
		{},
	}
	if !reflect.DeepEqual(arrayGot, arrayWant) {
		t.Errorf(" array not equal, want=%v, got=%v", arrayGot, arrayWant)
	}

	w.update(timeoutErrorOutcome)
	snapshotGot = w.getSnapshot()
	snapshotWant = newSnapshot(&counter{TotalCount: 4, ErrorCount: 2, TimeoutCount: 2})
	if !reflect.DeepEqual(snapshotGot, snapshotWant) {
		t.Errorf(" snapshot not equal, want=%v, got=%v", snapshotWant, snapshotGot)
	}
	arrayGot = w.array
	arrayWant = []*counter{
		{},
		{TotalCount: 1},
		{TotalCount: 1, ErrorCount: 1},
		{TotalCount: 1, TimeoutCount: 1},
		{TotalCount: 1, TimeoutCount: 1, ErrorCount: 1},
	}
	if !reflect.DeepEqual(arrayGot, arrayWant) {
		t.Errorf(" array not equal, want=%v, got=%v", arrayGot, arrayWant)
	}

	w.update(timeoutErrorOutcome)
	snapshotGot = w.getSnapshot()
	snapshotWant = newSnapshot(&counter{TotalCount: 5, ErrorCount: 3, TimeoutCount: 3})
	if !reflect.DeepEqual(snapshotGot, snapshotWant) {
		t.Errorf(" snapshot not equal, want=%v, got=%v", snapshotWant, snapshotGot)
	}
	arrayGot = w.array
	arrayWant = []*counter{
		{TotalCount: 1, TimeoutCount: 1, ErrorCount: 1},
		{TotalCount: 1},
		{TotalCount: 1, ErrorCount: 1},
		{TotalCount: 1, TimeoutCount: 1},
		{TotalCount: 1, TimeoutCount: 1, ErrorCount: 1},
	}
	if !reflect.DeepEqual(arrayGot, arrayWant) {
		t.Errorf(" array not equal, want=%v, got=%v", arrayGot, arrayWant)
	}

	w.update(timeoutErrorOutcome)
	snapshotGot = w.getSnapshot()
	snapshotWant = newSnapshot(&counter{TotalCount: 5, ErrorCount: 4, TimeoutCount: 4})
	if !reflect.DeepEqual(snapshotGot, snapshotWant) {
		t.Errorf(" snapshot not equal, want=%v, got=%v", snapshotWant, snapshotGot)
	}
	arrayGot = w.array
	arrayWant = []*counter{
		{TotalCount: 1, TimeoutCount: 1, ErrorCount: 1},
		{TotalCount: 1, TimeoutCount: 1, ErrorCount: 1},
		{TotalCount: 1, ErrorCount: 1},
		{TotalCount: 1, TimeoutCount: 1},
		{TotalCount: 1, TimeoutCount: 1, ErrorCount: 1},
	}
	if !reflect.DeepEqual(arrayGot, arrayWant) {
		t.Errorf(" array not equal, want=%v, got=%v", arrayGot, arrayWant)
	}

	w.update(timeoutErrorOutcome)
	snapshotGot = w.getSnapshot()
	snapshotWant = newSnapshot(&counter{TotalCount: 5, ErrorCount: 4, TimeoutCount: 5})
	if !reflect.DeepEqual(snapshotGot, snapshotWant) {
		t.Errorf(" snapshot not equal, want=%v, got=%v", snapshotWant, snapshotGot)
	}
	arrayGot = w.array
	arrayWant = []*counter{
		{TotalCount: 1, TimeoutCount: 1, ErrorCount: 1},
		{TotalCount: 1, TimeoutCount: 1, ErrorCount: 1},
		{TotalCount: 1, TimeoutCount: 1, ErrorCount: 1},
		{TotalCount: 1, TimeoutCount: 1},
		{TotalCount: 1, TimeoutCount: 1, ErrorCount: 1},
	}
	if !reflect.DeepEqual(arrayGot, arrayWant) {
		t.Errorf(" array not equal, want=%v, got=%v", arrayGot, arrayWant)
	}
}

func TestNewSnapshot(t *testing.T) {
	totals := &counter{
		TotalCount:   100,
		ErrorCount:   50,
		TimeoutCount: 40,
	}

	expected := WindowSnapshot{
		TotalCount:     100,
		SuccessCount:   50,
		ErrorCount:     50,
		ErrorPercent:   50,
		TimeoutCount:   40,
		TimeoutPercent: 40,
	}
	snapshot := newSnapshot(totals)

	if !reflect.DeepEqual(snapshot, expected) {
		t.Errorf("snapshot = %v, actual %v", snapshot, expected)
	}
}

//nolint:funlen
func Test_circleArray_moveWindowToCurrentSecond_slideOnceFromInit(t *testing.T) {
	type outcome struct {
		bucketStartTime int64
		endIndex        int
	}
	tests := []struct {
		name string
		ca   *timeWindowMetric
		now  int64
		want outcome
	}{
		{
			name: "start:0,now:4",
			ca:   newTimeWindowMetricWithTime(5, 0),
			now:  4,
			want: outcome{
				bucketStartTime: 4,
				endIndex:        3,
			},
		},
		{
			name: "start:1,now:3",
			ca:   newTimeWindowMetricWithTime(5, 1),
			now:  3,
			want: outcome{
				bucketStartTime: 3,
				endIndex:        1,
			},
		},
		{
			name: "start:2,now:6,move end index, no resets",
			ca:   newTimeWindowMetricWithTime(5, 2),
			now:  6,
			want: outcome{
				bucketStartTime: 6,
				endIndex:        3,
			},
		},
		{
			name: "start:0,now:6,dist curr to head > cap, reset all buckets",
			ca:   newTimeWindowMetricWithTime(5, 0),
			now:  6,
			want: outcome{
				bucketStartTime: 6,
				endIndex:        4,
			},
		},
		{
			name: "start:0,now:0,same bucket",
			ca:   newTimeWindowMetricWithTime(5, 0),
			now:  0,
			want: outcome{
				bucketStartTime: 0,
				endIndex:        4,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ca := tt.ca
			bkt := ca.moveWindowToCurrentSecond(tt.now)
			got := outcome{
				bucketStartTime: bkt.startTime,
				endIndex:        ca.endIndex,
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, windowOutcome %v, (types: %T vs %T)", got, tt.want, got, tt.want)
			}
		})
	}
}

func Test_circleArray_moveWindowToCurrentSecond_slideSameWindowConsecutively(t *testing.T) {
	type outcome struct {
		bucketStartTime int64
		endIndex        int
	}
	loops := []struct {
		name string
		now  int64
		want outcome
	}{
		{
			name: "now:4,move endIndex to 2",
			now:  4,
			want: outcome{
				bucketStartTime: 4,
				endIndex:        1,
			},
		},
		{
			name: "now:7, move end index by 3, rest 3 buckets",
			now:  7,
			want: outcome{
				bucketStartTime: 7,
				endIndex:        4,
			},
		},
		{
			name: "now:7, same bucket",
			now:  7,
			want: outcome{
				bucketStartTime: 7,
				endIndex:        4,
			},
		},
		{
			name: "now:8, move end index by 1, reset 1 bucket",
			now:  8,
			want: outcome{
				bucketStartTime: 8,
				endIndex:        0,
			},
		},
	}

	// init 5 buckets with start time of first bucket = 2
	// | -2 | -1 | 0 | 1 | 2 |
	ca := newTimeWindowMetricWithTime(5, 2)

	t.Run("test_slide_consecutively", func(t *testing.T) {
		for _, loop := range loops {
			bucket := ca.moveWindowToCurrentSecond(loop.now)

			got := outcome{
				bucketStartTime: bucket.startTime,
				endIndex:        ca.endIndex,
			}
			if !reflect.DeepEqual(got, loop.want) {
				t.Errorf("got %v, windowOutcome %v, (types: %T vs %T)", got, loop.want, got, loop.want)
			}
		}
	})
}
