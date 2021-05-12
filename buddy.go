// Copyright 2021. Silvano DAL ZILIO.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package rudd

import (
	"log"
	"sync/atomic"
)

// buddy implements a Binary Decision Diagram using the data structures and
// algorithms found in the BuDDy library.
type buddy struct {
	nodes           []bddNode   // List of all the BDD nodes. Constants are always kept at index 0 and 1
	freenum         int         // Number of free nodes
	freepos         int         // First free node
	varnum          int32       // number of BDD variables
	varset          [][2]int    // Set of variables used: we have a pair for each variable for its positive and negative occurrence
	refstack        []int       // Internal node reference stack
	error                       // Error status to help chain operations
	nodefinalizer   interface{} // Finalizer used to decrement the ref count of external references
	maxnodesize     int         // Maximum total number of nodes (0 if no limit)
	maxnodeincrease int         // Maximum number of nodes that can be added to the table at each resize (0 if no limit)
	minfreenodes    int         // Minimum number of nodes that should be left after GC before triggering a resize
	quantset        []int32     // Current variable set for quant.
	quantsetID      int32       // Current id used in quantset
	quantlast       int32       // Current last variable to be quant.
	bddStats                    // Information about the BDD
	gchistory       []gcStat    // Information about garbage collections
	cacheStat                   // Information about the caches
	*applycache                 // Cache for apply results
	*itecache                   // Cache for ITE results
	*quantcache                 // Cache for exist/forall results
	*appexcache                 // Cache for AppEx results
	*replacecache               // Cache for Replace results
}

// ************************************************************

// bddStats stores status information about a BDD.
type bddStats struct {
	produced         int    // Total number of new nodes ever produced
	setfinalizers    uint64 // Total number of external references to BDD nodes since the last GC
	calledfinalizers uint64 // Number of external references that were freed since the last GC
}

// ************************************************************

// Buddy initializes a new BDD implementing the algorithms in the BuDDy library.
// Parameter *nodesize* is the initial number of nodes in the nodetable and
// *cachesize* is the fixed size of the internal caches. Typical values for
// *nodesize* are 10 000 nodes for small test examples and up to 1 000 000 nodes
// for large examples. A cache size of 10 000 seems to work good even for large
// examples, but lesser values should do it for smaller examples.
//
// The number of cache entries can also be set to depend on the size of the
// nodetable using a call to *SetCacheRatio*.
//
// The initial number of nodes is not critical since the table will be resized
// whenever there are too few nodes left after a garbage collection. But it does
// have some impact on the efficency of the operations.
func Buddy(nodesize int, cachesize int) Set {
	b := &buddy{}
	nodesize = bdd_prime_gte(nodesize)
	b.minfreenodes = _MINFREENODES
	b.maxnodeincrease = _DEFAULTMAXNODEINC
	// initializing the list of nodes
	b.nodes = make([]bddNode, nodesize)
	for k := range b.nodes {
		b.nodes[k] = bddNode{
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
	b.cacheinit(cachesize)
	b.freepos = 2
	b.freenum = nodesize - 2
	b.varnum = 0
	b.gchistory = make([]gcStat, 0)
	b.maxnodeincrease = _DEFAULTMAXNODEINC
	b.error = nil
	b.nodefinalizer = func(n *int) {
		if _DEBUG {
			atomic.AddUint64(&(b.calledfinalizers), 1)
			if _LOGLEVEL > 2 {
				log.Printf("dec refcou %d\n", *n)
			}
		}
		b.nodes[*n].refcou--
	}
	return Set{b}
}

// ************************************************************

// True returns the constant true BDD
func (b *buddy) True() Node {
	return bddone
}

// False returns the constant false BDD
func (b *buddy) False() Node {
	return bddzero
}

// From returns a (constant) Node from a boolean value.
func (b *buddy) From(v bool) Node {
	if v {
		return bddone
	}
	return bddzero
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
