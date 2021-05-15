// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd

import (
	"bytes"
	"fmt"
	"log"
	"sync/atomic"
	"unsafe"
)

// hudd implements a Binary Decision Diagram using the runtime hashmap. We hash
// a triplet (level, low, high) to a string (we use a bytes.Buffer to avoid
// allocations) and use the unique table to associate an entry in the nodes
// table. We use more space but a benefit is that we can easily migrate to a
// concurrency-safe hashmap if we want to test concurrent data structures.
type hudd struct {
	bdd
	nodes         []huddnode     // List of all the BDD nodes. Constants are always kept at index 0 and 1
	unique        map[string]int // Unicity table, used to associate each triplet to a single node
	hbuff         bytes.Buffer   // Used to compute the hash of nodes. A Buffer needs no initialization.
	nodefinalizer interface{}    // Finalizer used to decrement the ref count of external references
	freenum       int            // Number of free nodes
	freepos       int            // First free node
	cacheStat                    // Information about the caches
}

type huddnode struct {
	level  int32 // Order of the variable in the BDD
	low    int   // Reference to the false branch
	high   int   // Reference to the true branch
	refcou int32 // Count the number of external references
}

func (b *hudd) ismarked(n int) bool {
	return (b.nodes[n].refcou & 0x200000) != 0
}

func (b *hudd) marknode(n int) {
	b.nodes[n].refcou |= 0x200000
}

func (b *hudd) unmarknode(n int) {
	b.nodes[n].refcou &= 0x1FFFFF
}

// Hudd initializes a new BDD implemented using the standard runtime hashmap.
// Options are similar to the case of the Buddy implementation.
func Hudd(varnum int, nodesize int, cachesizes ...int) Set {
	b := &hudd{}
	if (varnum < 1) || (varnum > int(_MAXVAR)) {
		b.seterror("bad number of variable (%d)", varnum)
		return Set{b}
	}
	b.varnum = int32(varnum)

	if _LOGLEVEL > 0 {
		log.Printf("set varnum to %d\n", b.varnum)
	}

	if nodesize < 2*varnum+2 {
		nodesize = 2*varnum + 2
	}
	b.minfreenodes = _MINFREENODES
	b.maxnodeincrease = _DEFAULTMAXNODEINC
	// initializing the list of nodes
	b.nodes = make([]huddnode, nodesize)
	for k := range b.nodes {
		b.nodes[k] = huddnode{
			level:  0,
			low:    -1,
			high:   k + 1,
			refcou: 0,
		}
	}
	b.nodes[nodesize-1].high = 0
	b.unique = make(map[string]int, nodesize)
	// creating bddzero and bddone. We do not add them to the unique table.
	b.nodes[0] = huddnode{
		level:  b.varnum,
		low:    0,
		high:   0,
		refcou: _MAXREFCOUNT,
	}
	b.nodes[1] = huddnode{
		level:  b.varnum,
		low:    1,
		high:   1,
		refcou: _MAXREFCOUNT,
	}
	b.freepos = 2
	// setting the last fields of b
	cachesize := 0
	cacheratio := 0
	if len(cachesizes) >= 1 {
		cachesize = cachesizes[0]
	}
	if len(cachesizes) >= 2 {
		cacheratio = cachesizes[1]
	}
	b.cacheinit(cachesize, cacheratio)

	b.varset = make([][2]int, varnum)
	// We also initialize the refstack.
	b.refstack = make([]int, 0, 2*varnum+4)
	b.initref()
	for k := int32(0); k < b.varnum; k++ {
		v0 := b.makenode(k, 0, 1)
		if v0 < 0 {
			b.seterror("cannot allocate new variable %d in setVarnum; %s", b.varnum, b.error)
			return Set{b}
		}
		b.nodes[v0].refcou = _MAXREFCOUNT
		b.pushref(v0)
		v1 := b.makenode(k, 1, 0)
		if v1 < 0 {
			b.seterror("cannot allocate new variable %d in setVarnum; %s", b.varnum, b.error)
			return Set{b}
		}
		b.nodes[v1].refcou = _MAXREFCOUNT
		b.popref(1)
		b.varset[k] = [2]int{v0, v1}
	}

	// We also need to resize the quantification cache
	b.quantset = make([]int32, b.varnum)
	b.quantsetID = 0

	b.gcstat.history = []gcpoint{}
	b.maxnodeincrease = _DEFAULTMAXNODEINC
	b.error = nil
	b.nodefinalizer = func(n *int) {
		if _DEBUG {
			atomic.AddUint64(&(b.gcstat.calledfinalizers), 1)
			if _LOGLEVEL > 2 {
				log.Printf("dec refcou %d\n", *n)
			}
		}
		b.nodes[*n].refcou--
	}
	return Set{b}
}

func (b *hudd) nodehash(level int32, low, high int) (int, bool) {
	b.hbuff.Reset()
	fmt.Fprintf(&b.hbuff, "%v %v %v", level, low, high)
	hn, ok := b.unique[b.hbuff.String()]
	return hn, ok
}

// When a slot is unused in b.nodes, we have low set to -1 and high set to the
// next free position. The value of b.freepos gives the index of the lowest
// unused slot, except when freenum is 0, in which case it is also 0.

func (b *hudd) setnode(level int32, low int, high int, count int32) int {
	b.hbuff.Reset()
	fmt.Fprintf(&b.hbuff, "%v %v %v", level, low, high)
	b.freenum--
	b.unique[b.hbuff.String()] = b.freepos
	res := b.freepos
	b.freepos = b.nodes[b.freepos].high
	b.nodes[res] = huddnode{level, low, high, count}
	return res
}

func (b *hudd) delnode(hn huddnode) {
	b.hbuff.Reset()
	fmt.Fprintf(&b.hbuff, "%v %v %v", hn.level, hn.low, hn.high)
	delete(b.unique, b.hbuff.String())
}

// Ithvar returns a BDD representing the i'th variable on success, otherwise we
// set the error status in the BDD and returns the constant False. The requested
// variable must be in the range [0..Varnum).
func (b *hudd) Ithvar(i int) Node {
	if (i < 0) || (int32(i) >= b.varnum) {
		b.seterror("Unknown variable used (%d) in call to ithvar", i)
		return bddzero
	}
	// we do not need to reference count variables
	return inode(b.varset[i][0])
}

// NIthvar returns a bdd representing the negation of the i'th variable on
// success, otherwise the constant false bdd. See *ithvar* for further info.
func (b *hudd) NIthvar(i int) Node {
	if (i < 0) || (int32(i) >= b.varnum) {
		return b.seterror("Unknown variable used (%d) in call to nithvar", i)
	}
	// we do not need to reference count variables
	return inode(b.varset[i][1])
}

// Label returns the variable (index) corresponding to node n in the BDD. We set
// the BDD to its error state and return -1 if we try to access a constant node.
func (b *hudd) Label(n Node) int {
	if b.checkptr(n) != nil {
		b.seterror("Illegal access to node %d in call to Label", n)
		return -1
	}
	if *n < 2 {
		b.seterror("Try to access label of constant node")
		return -1
	}
	return int(b.nodes[*n].level)
}

// Low returns the false branch of a BDD. We return bdderror if there is an
// error and set the error flag in the BDD.
func (b *hudd) Low(n Node) Node {
	if b.checkptr(n) != nil {
		return b.seterror("Illegal access to node %d in call to Low", n)
	}
	return b.retnode(b.nodes[*n].low)
}

// High returns the true branch of a BDD. We return bdderror if there is an
// error and set the error flag in the BDD.
func (b *hudd) High(n Node) Node {
	if b.checkptr(n) != nil {
		return b.seterror("Illegal access to node %d in call to High", n)
	}
	return b.retnode(b.nodes[*n].high)
}

// Stats returns information about the BDD
func (b *hudd) Stats() string {
	res := "==============\n"
	res += fmt.Sprintf("Varnum:     %d\n", b.varnum)
	res += fmt.Sprintf("Allocated:  %d\n", len(b.nodes))
	res += fmt.Sprintf("Produced:   %d\n", b.produced)
	r := (float64(b.freenum) / float64(len(b.nodes))) * 100
	res += fmt.Sprintf("Free:       %d  (%.3g %%)\n", b.freenum, r)
	res += fmt.Sprintf("Used:       %d  (%.3g %%)\n", len(b.nodes)-b.freenum, (100.0 - r))
	res += fmt.Sprintf("Size:       %s\n", humanSize(len(b.nodes), unsafe.Sizeof(buddynode{})))
	res += b.gcstats()
	if _DEBUG {
		res += "==============\n"
		res += b.cacheStat.String()
		res += b.applycache.String()
		res += b.itecache.String()
		res += b.quantcache.String()
		res += b.appexcache.String()
		res += b.replacecache.String()
	}
	return res
}
