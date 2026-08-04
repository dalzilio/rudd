// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd

import "sync/atomic"

// retain records an external reference while preventing node-table
// reclamation or resizing from moving the referenced slot.
func (b *tables) retain(n int) (retained bool, valid bool) {
	b.lifecycleMu.RLock()
	defer b.lifecycleMu.RUnlock()
	if n < 0 || n >= len(b.nodes) {
		return false, false
	}

	refcou := &b.nodes[n].refcou
	for {
		current := atomic.LoadInt32(refcou)
		if current&_MAXREFCOUNT == _MAXREFCOUNT {
			return false, true
		}
		if atomic.CompareAndSwapInt32(refcou, current, current+1) {
			return true, true
		}
	}
}

// release drops a counted external reference. Saturated references stay
// saturated so an uncounted reference can never become collectible.
func (b *tables) release(n int) {
	b.lifecycleMu.RLock()
	defer b.lifecycleMu.RUnlock()
	if n < 0 || n >= len(b.nodes) {
		return
	}

	refcou := &b.nodes[n].refcou
	for {
		current := atomic.LoadInt32(refcou)
		count := current & _MAXREFCOUNT
		if count == 0 || count == _MAXREFCOUNT {
			return
		}
		if atomic.CompareAndSwapInt32(refcou, current, current-1) {
			return
		}
	}
}

func (b *tables) hasReferences(n int) bool {
	return atomic.LoadInt32(&b.nodes[n].refcou)&_MAXREFCOUNT != 0
}
