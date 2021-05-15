// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd

import (
	"log"
	"math"
	"runtime"
	"sync/atomic"
)

func (b *hudd) retnode(n int) Node {
	if n < 0 || n > len(b.nodes) {
		if _DEBUG {
			log.Print(b.Error())
			log.Panicf("b.retnode(%d) not valid\n", n)
		}
		return nil
	}
	if n == 0 {
		return bddzero
	}
	if n == 1 {
		return bddone
	}
	x := n
	if b.nodes[n].refcou < _MAXREFCOUNT {
		b.nodes[n].refcou++
		runtime.SetFinalizer(&x, b.nodefinalizer)
		if _DEBUG {
			atomic.AddUint64(&(b.setfinalizers), 1)
			if _LOGLEVEL > 2 {
				log.Printf("inc refcou %d\n", n)
			}
		}
	}
	return &x
}

func (b *hudd) makenode(level int32, low int, high int) int {
	if _DEBUG {
		b.cacheStat.uniqueAccess++
	}
	// check whether childen are equal, in which case we can skip the node
	if low == high {
		return low
	}
	// otherwise try to find an existing node using the unique table
	if res, ok := b.nodehash(level, low, high); ok {
		if _DEBUG {
			b.cacheStat.uniqueHit++
		}
		return res
	}
	if _DEBUG {
		b.cacheStat.uniqueMiss++
	}
	// If no existing node, we build one. If there is no available spot
	// (b.freepos == 0), we try garbage collection and, as a last resort,
	// resizing the BDD list.
	if b.freepos == 0 {
		// We garbage collect unused nodes to try and find spare space.
		b.gbc()
		// We also test if we are under the threshold for resising.
		if (b.freenum*100)/len(b.nodes) <= b.minfreenodes {
			if err := b.noderesize(); err != nil {
				b.seterror("Unable to free memory or resize BDD")
				return -1
			}
		}
		// Panic if we still have no free positions after all this
		if b.freepos == 0 {
			b.seterror("Unable to resize BDD")
			return -1
		}
	}
	// We can now build the new node in the first available spot
	b.produced++
	return b.setnode(level, low, high, 0)
}

func (b *hudd) gbc() {
	if _LOGLEVEL > 0 {
		log.Println("starting GC")
	}

	if b.error != nil {
		return
	}

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
	}
	b.freepos = 0
	b.freenum = 0
	// we do a pass through the nodes list to void the unmarked nodes. After
	// finishing this pass, b.freepos points to the first free position in
	// b.nodes, or it is 0 if we found none.
	for n := len(b.nodes) - 1; n > 1; n-- {
		if b.ismarked(n) && (b.nodes[n].low != -1) {
			b.unmarknode(n)
		} else {
			b.delnode(b.nodes[n])
			b.nodes[n].low = -1
			b.nodes[n].high = b.freepos
			b.freepos = n
			b.freenum++
		}
	}
	// we also invalidate the caches
	b.cachereset()
	if _LOGLEVEL > 0 {
		log.Printf("end GC; freenum: %d\n", b.freenum)
	}
}

func (b *hudd) noderesize() error {
	if _LOGLEVEL > 0 {
		log.Printf("start resize: %d\n", len(b.nodes))
	}
	if b.error != nil {
		b.seterror("Error before resizing; %s", b.error)
		return b.error
	}
	oldsize := len(b.nodes)
	nodesize := len(b.nodes)
	if (oldsize >= b.maxnodesize) && (b.maxnodesize > 0) {
		b.seterror("Cannot resize BDD, already at max capacity (%d nodes)", b.maxnodesize)
		return b.error
	}
	if oldsize > (math.MaxInt32 >> 1) {
		nodesize = math.MaxInt32 - 1
	} else {
		nodesize = nodesize << 1
	}
	if nodesize > (oldsize + b.maxnodeincrease) {
		nodesize = oldsize + b.maxnodeincrease
	}
	if (nodesize > b.maxnodesize) && (b.maxnodesize > 0) {
		nodesize = b.maxnodesize
	}
	if nodesize <= oldsize {
		b.seterror("Unable to grow size of BDD (%d nodes)", nodesize)
		return b.error
	}

	tmp := b.nodes
	b.nodes = make([]huddnode, nodesize)
	copy(b.nodes, tmp)
	tmp = nil

	for n := oldsize; n < nodesize; n++ {
		b.nodes[n].refcou = 0
		b.nodes[n].level = 0
		b.nodes[n].low = -1
		b.nodes[n].high = n + 1
	}
	b.nodes[nodesize-1].high = b.freepos
	b.freepos = oldsize
	b.freenum += (nodesize - oldsize)

	b.cacheresize(len(b.nodes))

	if _LOGLEVEL > 0 {
		log.Printf("end resize: %d\n", len(b.nodes))
	}

	return nil
}

func (b *hudd) markrec(n int) {
	if n < 2 || b.ismarked(n) || (b.nodes[n].low == -1) {
		return
	}
	b.marknode(n)
	b.markrec(b.nodes[n].low)
	b.markrec(b.nodes[n].high)
}

func (b *hudd) unmarkall() {
	for k, v := range b.nodes {
		if k < 2 || !b.ismarked(k) || (v.low == -1) {
			continue
		}
		b.unmarknode(k)
	}
}

// Scanset returns the set of variables (levels) found when following the high
// branch of node n. This is the dual of function Makeset. The result may be nil
// if there is an error. The result is not necessarily sorted (but follows the
// level order).
func (b *hudd) Scanset(n Node) []int {
	if b.checkptr(n) != nil {
		return nil
	}
	if *n < 2 {
		return nil
	}
	res := []int{}
	for i := *n; i > 1; i = b.nodes[i].high {
		res = append(res, int(b.nodes[i].level))
	}
	return res
}

// Makeset returns a node corresponding to the conjunction (the cube) of all the
// variable in varset, in their positive form. It is such that
// scanset(Makeset(a)) == a. It returns False and sets the error condition in b
// if one of the variables is outside the scope of the BDD (see documentation
// for function *Ithvar*).
func (b *hudd) Makeset(varset []int) Node {
	res := bddone
	for _, level := range varset {
		// FIXME: should find a way to do it without adding references
		tmp := b.Apply(res, b.Ithvar(level), OPand)
		if b.error != nil {
			return bddzero
		}
		res = tmp
	}
	return res
}