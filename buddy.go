// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd

import (
	"fmt"
	"log"
	"sync/atomic"
	"unsafe"
)

// buddy implements a Binary Decision Diagram using the data structures and
// algorithms found in the BuDDy library.
type buddy struct {
	nodes         []buddynode // List of all the BDD nodes. Constants are always kept at index 0 and 1
	freenum       int         // Number of free nodes
	freepos       int         // First free node
	produced      int         // Total number of new nodes ever produced
	nodefinalizer interface{} // Finalizer used to decrement the ref count of external references
	uniqueAccess  int         // accesses to the unique node table
	uniqueChain   int         // iterations through the cache chains in the unique node table
	uniqueHit     int         // entries actually found in the the unique node table
	uniqueMiss    int         // entries not found in the the unique node table
	gcstat                    // Information about garbage collections
	configs                   // Configurable parameters
}

type buddynode struct {
	refcou int32 // Count the number of external references
	level  int32 // Order of the variable in the BDD
	low    int   // Reference to the false branch
	high   int   // Reference to the true branch
	hash   int   // Index where to (possibly) find node with this hash value
	next   int   // Next index to check in case of a collision, 0 if last
}

func (b *buddy) ismarked(n int) bool {
	return (b.nodes[n].level & 0x200000) != 0
}

func (b *buddy) marknode(n int) {
	b.nodes[n].level = b.nodes[n].level | 0x200000
}

func (b *buddy) unmarknode(n int) {
	b.nodes[n].level = b.nodes[n].level & 0x1FFFFF
}

// Buddy initializes a new BDD implementing the algorithms in the BuDDy library,
// wher varnum is the number of variables in the BDD, and nodesize is the size
// of the initial node table. Typical values for nodesize are 10 000 nodes for
// small test examples and up to 1 000 000 nodes for large examples.
//
// You can specify optional (int) parameters. Values after the first two
// optional parameters will not be taken into account. The first value is to
// specify a cachesize for the internal caches. (A cache size of 10 000 seems to
// work good even for large examples, but lesser values should do it for smaller
// examples.) A second extra value is used to set a "cache ratio" so that caches
// can grow each time we resize the node table. With a cache ratio of r, there
// is one available slot in the cache for every r slots in the node table. (A
// typical value for the cache ratio is 4 or 5).  A cache ratio of 0 (the
// default) means that the cache size is fixed.
//
// The initial number of nodes is not critical since the table will be resized
// whenever there are too few nodes left after a garbage collection. But it does
// have some impact on the efficency of the operations.
func Buddy(varnum int, nodesize int, cachesizes ...int) Set {
	b := &bdd{}
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
	cachesize := 0
	cacheratio := 0
	if len(cachesizes) >= 1 {
		cachesize = cachesizes[0]
	}
	if len(cachesizes) >= 2 {
		cacheratio = cachesizes[1]
	}
	b.varset = make([][2]int, varnum)
	// We also initialize the refstack.
	b.refstack = make([]int, 0, 2*varnum+4)
	b.initref()
	b.error = nil
	b.implementation = makebuddy(nodesize, b)
	b.cacheinit(cachesize, cacheratio)
	return Set{b}
}

func makebuddy(nodesize int, config *bdd) *buddy {
	// initializing the list of nodes
	b := &buddy{}
	b.minfreenodes = _MINFREENODES
	b.maxnodeincrease = _DEFAULTMAXNODEINC
	nodesize = bdd_prime_gte(nodesize)
	b.nodes = make([]buddynode, nodesize)
	for k := range b.nodes {
		b.nodes[k] = buddynode{
			refcou: 0,
			level:  0,
			low:    -1,
			high:   0,
			hash:   0,
			next:   k + 1,
		}
	}
	b.nodes[nodesize-1].next = 0
	b.nodes[0].refcou = _MAXREFCOUNT
	b.nodes[1].refcou = _MAXREFCOUNT
	b.nodes[0].low = 0
	b.nodes[0].high = 0
	b.nodes[1].low = 1
	b.nodes[1].high = 1
	b.nodes[0].level = config.varnum
	b.nodes[1].level = config.varnum
	b.freepos = 2
	b.freenum = nodesize - 2
	b.gcstat.history = []gcpoint{}
	b.nodefinalizer = func(n *int) {
		if _DEBUG {
			atomic.AddUint64(&(b.gcstat.calledfinalizers), 1)
			if _LOGLEVEL > 2 {
				log.Printf("dec refcou %d\n", *n)
			}
		}
		b.nodes[*n].refcou--
	}
	for k := int32(0); k < config.varnum; k++ {
		v0, _ := b.makenode(k, 0, 1, nil)
		if v0 < 0 {
			config.seterror("cannot allocate new variable %d in setVarnum", k)
			return nil
		}
		b.nodes[v0].refcou = _MAXREFCOUNT
		config.pushref(v0)
		v1, _ := b.makenode(k, 1, 0, nil)
		if v1 < 0 {
			config.seterror("cannot allocate new variable %d in setVarnum", k)
			return nil
		}
		b.nodes[v1].refcou = _MAXREFCOUNT
		config.popref(1)
		config.varset[k] = [2]int{v0, v1}
	}
	return b
}

func (b *buddy) size() int {
	return len(b.nodes)
}

func (b *buddy) level(n int) int32 {
	return b.nodes[n].level
}

func (b *buddy) low(n int) int {
	return b.nodes[n].low
}

func (b *buddy) high(n int) int {
	return b.nodes[n].high
}

func (b *buddy) allnodesfrom(f func(id, level, low, high int) error, n []Node) error {
	for _, v := range n {
		b.markrec(*v)
	}
	if err := f(0, int(b.nodes[0].level), 0, 0); err != nil {
		b.unmarkall()
		return err
	}
	if err := f(1, int(b.nodes[1].level), 1, 1); err != nil {
		b.unmarkall()
		return err
	}
	for k := range b.nodes {
		if k > 1 && b.ismarked(k) {
			b.unmarknode(k)
			if err := f(k, int(b.nodes[k].level), b.nodes[k].low, b.nodes[k].high); err != nil {
				b.unmarkall()
				return err
			}
		}
	}
	return nil
}

func (b *buddy) allnodes(f func(id, level, low, high int) error) error {
	if err := f(0, int(b.nodes[0].level), 0, 0); err != nil {
		return err
	}
	if err := f(1, int(b.nodes[1].level), 1, 1); err != nil {
		return err
	}
	for k, v := range b.nodes {
		if v.low != -1 {
			if err := f(k, int(v.level), v.low, v.high); err != nil {
				return err
			}
		}
	}
	return nil
}

// Stats returns information about the BDD
func (b *buddy) stats() string {
	res := fmt.Sprintf("Allocated:  %d\n", len(b.nodes))
	res += fmt.Sprintf("Produced:   %d\n", b.produced)
	r := (float64(b.freenum) / float64(len(b.nodes))) * 100
	res += fmt.Sprintf("Free:       %d  (%.3g %%)\n", b.freenum, r)
	res += fmt.Sprintf("Used:       %d  (%.3g %%)\n", len(b.nodes)-b.freenum, (100.0 - r))
	res += fmt.Sprintf("Size:       %s\n", humanSize(len(b.nodes), unsafe.Sizeof(buddynode{})))
	res += "==============\n"
	res += fmt.Sprintf("# of GC:    %d\n", len(b.gcstat.history))
	if _DEBUG {
		allocated := int(b.gcstat.setfinalizers)
		reclaimed := int(b.gcstat.calledfinalizers)
		for _, g := range b.gcstat.history {
			allocated += g.setfinalizers
			reclaimed += g.calledfinalizers
		}
		res += fmt.Sprintf("Ext. refs:  %d\n", allocated)
		res += fmt.Sprintf("Reclaimed:  %d\n", reclaimed)
		res += "==============\n"
		res += fmt.Sprintf("Unique Access:  %d\n", b.uniqueAccess)
		res += fmt.Sprintf("Unique Chain:   %d\n", b.uniqueChain)
		res += fmt.Sprintf("Unique Hit:     %d\n", b.uniqueHit)
		res += fmt.Sprintf("Unique Miss:    %d\n", b.uniqueMiss)
	}
	return res
}
