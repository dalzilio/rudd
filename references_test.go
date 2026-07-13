// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

func TestReferenceCountSaturates(t *testing.T) {
	bdd, err := New(4, Nodesize(64))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	node := bdd.And(bdd.Ithvar(0), bdd.Ithvar(1))
	if node == nil || *node < 2 {
		t.Fatalf("And() node = %v, want a non-terminal node", node)
	}
	index := *node
	runtime.SetFinalizer(node, nil)
	bdd.release(index)

	for count := int32(0); count < _MAXREFCOUNT; count++ {
		retained, valid := bdd.retain(index)
		if !valid || !retained {
			t.Fatalf("retain() at count %d = (%t, %t), want (true, true)", count, retained, valid)
		}
	}
	if retained, valid := bdd.retain(index); !valid || retained {
		t.Fatalf("retain() at saturation = (%t, %t), want (false, true)", retained, valid)
	}

	bdd.release(index)
	if got := atomic.LoadInt32(&bdd.nodes[index].refcou) & _MAXREFCOUNT; got != _MAXREFCOUNT {
		t.Fatalf("reference count after saturated release = %d, want %d", got, _MAXREFCOUNT)
	}
}

func TestRetainRejectsOutOfRangeNode(t *testing.T) {
	bdd, err := New(2, Nodesize(16))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []struct {
		name  string
		index int
	}{
		{name: "negative", index: -1},
		{name: "length", index: len(bdd.nodes)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if retained, valid := bdd.retain(tt.index); valid || retained {
				t.Fatalf("retain(%d) = (%t, %t), want (false, false)", tt.index, retained, valid)
			}
			bdd.release(tt.index)
		})
	}
}

func TestBDDOperationsRemainStableDuringFinalization(t *testing.T) {
	bdd, err := New(4, Nodesize(256), Cachesize(128))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	left := bdd.And(bdd.Ithvar(0), bdd.NIthvar(1))
	right := bdd.And(bdd.NIthvar(0), bdd.Ithvar(1))
	wantUnion := bdd.Or(left, right)

	stopGC := make(chan struct{})
	var gcDone sync.WaitGroup
	gcDone.Add(1)
	go func() {
		defer gcDone.Done()
		for {
			select {
			case <-stopGC:
				return
			default:
				runtime.GC()
				runtime.Gosched()
			}
		}
	}()
	t.Cleanup(func() {
		close(stopGC)
		gcDone.Wait()
		runtime.KeepAlive(wantUnion)
		runtime.KeepAlive(left)
		runtime.KeepAlive(right)
		runtime.KeepAlive(bdd)
	})

	for iteration := 0; iteration < 200_000; iteration++ {
		union := bdd.Or(left, right)
		intersection := bdd.And(union, left)
		difference := bdd.Apply(union, right, OPdiff)
		if iteration%1_000 == 0 {
			if !bdd.Equal(union, wantUnion) {
				t.Fatalf("union changed at iteration %d", iteration)
			}
			if !bdd.Equal(intersection, left) {
				t.Fatalf("intersection changed at iteration %d", iteration)
			}
			if !bdd.Equal(difference, left) {
				t.Fatalf("difference changed at iteration %d", iteration)
			}
		}
	}
}
