// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd

import (
	"log"
	"sync/atomic"
)

// bdd is the structure shared by all implementations of BDD where we use
// integer as the key for Nodes.
type bdd struct {
	varnum   int32    // number of BDD variables
	varset   [][2]int // Set of variables used: we have a pair for each variable for its positive and negative occurrence
	refstack []int    // Internal node reference stack
	produced int      // Total number of new nodes ever produced
	error             // Error status to help chain operations
}

// buddy implements a Binary Decision Diagram using the data structures and
// algorithms found in the BuDDy library.
type buddy struct {
	bdd
	nodes           []buddyNode // List of all the BDD nodes. Constants are always kept at index 0 and 1
	freenum         int         // Number of free nodes
	freepos         int         // First free node
	nodefinalizer   interface{} // Finalizer used to decrement the ref count of external references
	maxnodesize     int         // Maximum total number of nodes (0 if no limit)
	maxnodeincrease int         // Maximum number of nodes that can be added to the table at each resize (0 if no limit)
	minfreenodes    int         // Minimum number of nodes that should be left after GC before triggering a resize
	quantset        []int32     // Current variable set for quant.
	quantsetID      int32       // Current id used in quantset
	quantlast       int32       // Current last variable to be quant.
	gcstat                      // Information about garbage collections
	cacheStat                   // Information about the caches
	applycache                  // Cache for apply results
	itecache                    // Cache for ITE results
	quantcache                  // Cache for exist/forall results
	appexcache                  // Cache for AppEx results
	replacecache                // Cache for Replace results
}

// ************************************************************

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
	b := &buddy{}
	if nodesize < 2*varnum+2 {
		nodesize = 2*varnum + 2
	}
	nodesize = bdd_prime_gte(nodesize)
	b.minfreenodes = _MINFREENODES
	b.maxnodeincrease = _DEFAULTMAXNODEINC
	// initializing the list of nodes
	b.nodes = make([]buddyNode, nodesize)
	for k := range b.nodes {
		b.nodes[k] = buddyNode{
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
	b.freepos = 2
	b.freenum = nodesize - 2
	b.setVarnum(varnum)
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

// ************************************************************

// setVarnum sets the number of BDD variables. We call this function only once
// during initialization and generate the list used for Ithvar and NIthvar.
func (b *buddy) setVarnum(num int) error {
	inum := int32(num)
	if (inum < 1) || (inum > _MAXVAR) {
		b.seterror("bad number of variable (%d) in setVarnum", inum)
		return b.error
	}
	b.varnum = inum
	// We create new slices for the fields related to the list of variables:
	// varset, level2var, var2level.
	b.varset = make([][2]int, inum)

	// Constants always have the highest level.
	b.nodes[0].level = inum
	b.nodes[1].level = inum

	// We also initialize the refstack.
	b.refstack = make([]int, 0, 2*inum+4)
	b.initref()
	for k := int32(0); k < inum; k++ {
		v0 := b.makenode(k, 0, 1)
		if v0 < 0 {
			b.seterror("cannot allocate new variable %d in setVarnum; %s", b.varnum, b.error)
			return b.error
		}
		b.pushref(v0)
		v1 := b.makenode(k, 1, 0)
		if v1 < 0 {
			b.seterror("cannot allocate new variable %d in setVarnum; %s", b.varnum, b.error)
			return b.error
		}
		b.popref(1)
		b.varset[k] = [2]int{v0, v1}
		b.nodes[b.varset[k][0]].refcou = _MAXREFCOUNT
		b.nodes[b.varset[k][1]].refcou = _MAXREFCOUNT
	}

	// We also need to resize the quantification cache
	b.quantset = make([]int32, b.varnum)
	b.quantsetID = 0

	if _LOGLEVEL > 0 {
		log.Printf("set varnum to %d\n", b.varnum)
	}
	return nil
}

// ************************************************************

// Ithvar returns a BDD representing the i'th variable on success, otherwise we
// set the error status in the BDD and returns the constant False. The requested
// variable must be in the range [0..Varnum).
func (b *buddy) Ithvar(i int) Node {
	if (i < 0) || (int32(i) >= b.varnum) {
		b.seterror("Unknown variable used (%d) in call to ithvar", i)
		return bddzero
	}
	// we do not need to reference count variables
	return inode(b.varset[i][0])
}

// NIthvar returns a bdd representing the negation of the i'th variable on
// success, otherwise the constant false bdd. See *ithvar* for further info.
func (b *buddy) NIthvar(i int) Node {
	if (i < 0) || (int32(i) >= b.varnum) {
		return b.seterror("Unknown variable used (%d) in call to nithvar", i)
	}
	// we do not need to reference count variables
	return inode(b.varset[i][1])
}

// Varnum returns the number of defined variables.
func (b *buddy) Varnum() int {
	return int(b.varnum)
}

// Label returns the variable (index) corresponding to node n in the BDD. We set
// the BDD to its error state and return -1 if we try to access a constant node.
func (b *buddy) Label(n Node) int {
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
func (b *buddy) Low(n Node) Node {
	if b.checkptr(n) != nil {
		return b.seterror("Illegal access to node %d in call to Low", n)
	}
	return b.retnode(b.nodes[*n].low)
}

// High returns the true branch of a BDD. We return bdderror if there is an
// error and set the error flag in the BDD.
func (b *buddy) High(n Node) Node {
	if b.checkptr(n) != nil {
		return b.seterror("Illegal access to node %d in call to High", n)
	}
	return b.retnode(b.nodes[*n].high)
}
