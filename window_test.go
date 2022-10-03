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
