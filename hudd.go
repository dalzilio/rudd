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

// hudd implements a Binary Decision Diagram using the runtime hashmap. We hash
// a triplet (level, low, high) to a string (we use a bytes.Buffer to avoid
// allocations) and use the unique table to associate an entry in the nodes
// table. We use more space but a benefit is that we can easily migrate to a
// concurrency-safe hashmap if we want to test concurrent data structures.
type hudd struct {
	nodes         []huddnode             // List of all the BDD nodes. Constants are always kept at index 0 and 1
	unique        map[[huddsize]byte]int // Unicity table, used to associate each triplet to a single node
	freenum       int                    // Number of free nodes
	freepos       int                    // First free node
	produced      int                    // Total number of new nodes ever produced
	hbuff         [huddsize]byte         // Used to compute the hash of nodes. A Buffer needs no initialization.
	nodefinalizer interface{}            // Finalizer used to decrement the ref count of external references
	uniqueAccess  int                    // accesses to the unique node table
	uniqueHit     int                    // entries actually found in the the unique node table
	uniqueMiss    int                    // entries not found in the the unique node table
	gcstat                               // Information about garbage collections
	configs                              // Configurable parameters
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
	b.implementation = makehudd(nodesize, b)
	b.cacheinit(cachesize, cacheratio)
	return Set{b}
}

func makehudd(nodesize int, config *bdd) *hudd {
	b := &hudd{}
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
	b.unique = make(map[[huddsize]byte]int, nodesize)
	// creating bddzero and bddone. We do not add them to the unique table.
	b.nodes[0] = huddnode{
		level:  config.varnum,
		low:    0,
		high:   0,
		refcou: _MAXREFCOUNT,
	}
	b.nodes[1] = huddnode{
		level:  config.varnum,
		low:    1,
		high:   1,
		refcou: _MAXREFCOUNT,
	}
	b.freepos = 2
	for k := int32(0); k < config.varnum; k++ {
		v0, _ := b.makenode(k, 0, 1, nil)
		if v0 < 0 {
			config.seterror("cannot allocate new variable %d in setVarnum", k)
			return b
		}
		b.nodes[v0].refcou = _MAXREFCOUNT
		config.pushref(v0)
		v1, _ := b.makenode(k, 1, 0, nil)
		if v1 < 0 {
			config.seterror("cannot allocate new variable %d in setVarnum", k)
			return b
		}
		b.nodes[v1].refcou = _MAXREFCOUNT
		config.popref(1)
		config.varset[k] = [2]int{v0, v1}
	}
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
	return b
}

func (b *hudd) huddhash(level int32, low, high int) {
	b.hbuff[0] = byte(level)
	b.hbuff[1] = byte(level >> 8)
	b.hbuff[2] = byte(level >> 16)
	b.hbuff[3] = byte(level >> 24)
	b.hbuff[4] = byte(low)
	b.hbuff[5] = byte(low >> 8)
	b.hbuff[6] = byte(low >> 16)
	b.hbuff[7] = byte(low >> 24)
	if huddsize == 20 {
		// 64 bits machine
		b.hbuff[8] = byte(low >> 32)
		b.hbuff[9] = byte(low >> 40)
		b.hbuff[10] = byte(low >> 48)
		b.hbuff[11] = byte(low >> 56)
		b.hbuff[12] = byte(high)
		b.hbuff[13] = byte(high >> 8)
		b.hbuff[14] = byte(high >> 16)
		b.hbuff[15] = byte(high >> 24)
		b.hbuff[16] = byte(high >> 32)
		b.hbuff[17] = byte(high >> 40)
		b.hbuff[18] = byte(high >> 48)
		b.hbuff[19] = byte(high >> 56)
		return
	}
	// 32 bits machine
	b.hbuff[8] = byte(high)
	b.hbuff[9] = byte(high >> 8)
	b.hbuff[10] = byte(high >> 16)
	b.hbuff[11] = byte(high >> 24)
}

func (b *hudd) nodehash(level int32, low, high int) (int, bool) {
	b.huddhash(level, low, high)
	hn, ok := b.unique[b.hbuff]
	return hn, ok
}

// When a slot is unused in b.nodes, we have low set to -1 and high set to the
// next free position. The value of b.freepos gives the index of the lowest
// unused slot, except when freenum is 0, in which case it is also 0.

func (b *hudd) setnode(level int32, low int, high int, count int32) int {
	b.huddhash(level, low, high)
	b.freenum--
	b.unique[b.hbuff] = b.freepos
	res := b.freepos
	b.freepos = b.nodes[b.freepos].high
	b.nodes[res] = huddnode{level, low, high, count}
	return res
}

func (b *hudd) delnode(hn huddnode) {
	b.huddhash(hn.level, hn.low, hn.high)
	delete(b.unique, b.hbuff)
}

func (b *hudd) size() int {
	return len(b.nodes)
}

func (b *hudd) level(n int) int32 {
	return b.nodes[n].level
}

func (b *hudd) low(n int) int {
	return b.nodes[n].low
}

func (b *hudd) high(n int) int {
	return b.nodes[n].high
}

func (b *hudd) allnodesfrom(f func(id, level, low, high int) error, n []Node) error {
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

func (b *hudd) allnodes(f func(id, level, low, high int) error) error {
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

// stats returns information about the implementation
func (b *hudd) stats() string {
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
		res += fmt.Sprintf("Unique Hit:     %d\n", b.uniqueHit)
		res += fmt.Sprintf("Unique Miss:    %d\n", b.uniqueMiss)
	}
	return res
}
