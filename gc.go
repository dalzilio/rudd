// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd

import (
	"log"
)

// gcstat stores status information about garbage collections. We use a stack
// (slice) of objects to record the sequence of GC during a computation.
type gcstat struct {
	setfinalizers    uint64    // Total number of external references to BDD nodes
	calledfinalizers uint64    // Number of external references that were freed
	history          []gcpoint // Snaphot of GC stats at each occurrence
}

type gcpoint struct {
	nodes            int // Total number of allocated nodes in the nodetable
	freenodes        int // Number of free nodes in the nodetable
	setfinalizers    int // Total number of external references to BDD nodes
	calledfinalizers int // Number of external references that were freed
}

// *************************************************************************

// AddRef increases the reference count on node n and returns n so that calls
// can be easily chained together. A call to AddRef can never raise an error,
// even if we access an unused node or a value outside the range of the BDD.
//
// Reference counting is done on externaly referenced nodes only and the count
// for a specific node can and must be increased using this function to avoid
// loosing the node during garbage collection.
func (b *buddy) AddRef(n Node) Node {
	if *n < 2 {
		return n
	}
	if *n >= len(b.nodes) {
		return n
	}
	if b.nodes[*n].low == -1 {
		return n
	}
	if b.nodes[*n].refcou < _MAXREFCOUNT {
		b.nodes[*n].refcou++
	}
	return n
}

// DelRef decreases the reference count on a node and returns n so that calls
// can be easily chained together. A call to DelRef can never raise an error,
// even if we access an unused node or a value outside the range of the BDD.
//
// Like with AddRef, reference counting is done on externaly referenced nodes
// only and the count for a specific node can and must be decreased using this
// function to make it possible to reclaim the node during garbage collection.
func (b *buddy) DelRef(n Node) Node {
	if *n >= len(b.nodes) {
		return n
	}
	if b.nodes[*n].low == -1 {
		return n
	}
	/* if the following line is present, fails there much earlier */
	if b.nodes[*n].refcou <= 0 {
		return n
	}
	if b.nodes[*n].refcou < _MAXREFCOUNT {
		b.nodes[*n].refcou--
	}
	return n
}

// *************************************************************************

// gbc is the garbage collector called for reclaiming memory, inside a call to
// makenode, when there are no free positions available. Allocated nodes that
// are not reclaimed do not move.
func (b *buddy) gbc() {
	if _LOGLEVEL > 0 {
		log.Println("starting GC")
		if _LOGLEVEL > 2 {
			b.logTable()
		}
	}

	if b.error != nil {
		return
	}

	// We could  explictly ask the system to run its GC so that we can decrement
	// the ref counts of Nodes that had an external reference. This is blocking.
	// Frequent GC is time consuming, but with fewer GC we can experience more
	// resizing events.
	//
	// runtime.GC()

	// we append the current stats to the GC history
	if _DEBUG {
		b.gcstat.history = append(b.gcstat.history, gcpoint{
			nodes:            len(b.nodes),
			freenodes:        b.freenum,
			setfinalizers:    int(b.gcstat.setfinalizers),
			calledfinalizers: int(b.gcstat.calledfinalizers),
		})
		b.gcstat.setfinalizers = 0
		b.gcstat.calledfinalizers = 0
		if _LOGLEVEL > 0 {
			log.Printf("runtime.GC() reclaimed %d references\n", b.gcstat.calledfinalizers)
		}
	} else {
		b.gcstat.history = append(b.gcstat.history, gcpoint{
			nodes:     len(b.nodes),
			freenodes: b.freenum,
		})
	}
	// we mark the nodes in the refstack to avoid collecting them
	for _, r := range b.refstack {
		b.markrec(int(r))
	}
	// we also protect nodes with a positive refcount (and therefore also the
	// ones with a MAXREFCOUNT, such has variables)
	for k := range b.nodes {
		if b.nodes[k].refcou > 0 {
			b.markrec(k)
		}
		b.nodes[k].hash = 0
	}
	b.freepos = 0
	b.freenum = 0
	// we do a pass through the nodes list to update the hash chains and void
	// the unmarked nodes. After finishing this pass, b.freepos points to the
	// first free position in b.nodes, or it is 0 if we found none.
	for n := len(b.nodes) - 1; n > 1; n-- {
		if b.ismarked(n) && (b.nodes[n].low != -1) {
			b.unmarknode(n)
			hash := b.ptrhash(int(n))
			b.nodes[n].next = b.nodes[hash].hash
			b.nodes[hash].hash = int(n)
		} else {
			b.nodes[n].low = -1
			b.nodes[n].next = b.freepos
			b.freepos = n
			b.freenum++
		}
	}
	// we also invalidate the caches
	b.cachereset()
	if _LOGLEVEL > 0 {
		log.Printf("end GC; freenum: %d\n", b.freenum)
		if _LOGLEVEL > 2 {
			b.logTable()
		}
	}
}

// *************************************************************************
// RECURSIVE MARK / UNMARK

func (b *buddy) markrec(n int) {
	if n < 2 || b.ismarked(n) || (b.nodes[n].low == -1) {
		return
	}
	b.marknode(n)
	b.markrec(b.nodes[n].low)
	b.markrec(b.nodes[n].high)
}

// func (b *BDD) mark_upto(n int, level int32) {
// 	if n < 2 {
// 		return
// 	}
// 	if b.ismarked(n) || (b.nodes[n].low == -1) {
// 		return
// 	}
// 	if b.nodes[n].level > level {
// 		return
// 	}
// 	b.marknode(n)
// 	b.mark_upto(b.nodes[n].low, level)
// 	b.mark_upto(b.nodes[n].high, level)
// }

// markcount returns the number of successors of the node n and mark them.
// func (b *buddy) markcount(n int) int {
// 	if n < 2 {
// 		return 0
// 	}
// 	if b.ismarked(n) || (b.nodes[n].low == -1) {
// 		return 0
// 	}
// 	b.marknode(n)
// 	return 1 + b.markcount(b.nodes[n].low) + b.markcount(b.nodes[n].high)
// }

func (b *buddy) unmarkall() {
	for k, v := range b.nodes {
		if k < 2 || !b.ismarked(k) || (v.low == -1) {
			continue
		}
		b.unmarknode(k)
	}
}

// func (b *BDD) unmark_upto(n int, level int32) {
// 	if n < 2 {
// 		return
// 	}
// 	if b.ismarked(n) || (b.nodes[n].low == int(-1)) {
// 		return
// 	}
// 	b.unmarknode(n)
// 	if b.nodes[n].level > level {
// 		return
// 	}
// 	b.unmark_upto(b.nodes[n].low, level)
// 	b.unmark_upto(b.nodes[n].high, level)
// }

// *************************************************************************
// private functions to manipulate the refstack; used to prevent nodes that are
// currently being built (e.g. transient nodes built during an apply) to be
// reclaimed during GC.

func (b *buddy) initref() {
	b.refstack = b.refstack[:0]
}

func (b *buddy) pushref(n int) int {
	b.refstack = append(b.refstack, n)
	return n
}

func (b *buddy) popref(a int) {
	b.refstack = b.refstack[:len(b.refstack)-a]
}
