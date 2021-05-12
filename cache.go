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
	"fmt"
	"math"
)

// ************************************************************
// cache is used for caching apply/exist etc. results
type cache struct {
	cacheratio int // value used to resize the caches as a factor of the number of nodes
	table      []cacheData
}

// cacheStat stores status information about cache usage
type cacheStat struct {
	uniqueAccess int // accesses to the unique node table
	uniqueChain  int // iterations through the cache chains in the unique node table
	uniqueHit    int // entries actually found in the the unique node table
	uniqueMiss   int // entries not found in the the unique node table
	opHit        int // entries found in the operator caches
	opMiss       int // entries not found in the operator caches
}

// cacheData is a unit of information stored in the Apply and ITE cache
type cacheData struct {
	res int
	a   int
	b   int
	c   int
}

// ************************************************************

// Different kind of caches used in the bdd

type applycache struct {
	cache          // Cache for apply results
	op    Operator // Current operation during an apply
}

type itecache struct {
	cache // Cache for ITE results
}

type quantcache struct {
	cache     // Cache for exist/forall results
	id    int // Current cache id for quantifications
}

// appexcache are a mix of  quant and apply caches
type appexcache struct {
	cache          // Cache for appex/appall results
	id    int      // Current cache id for quantifications
	op    Operator // Current operator for appex
}

type replacecache struct {
	cache     // Cache for replace results
	id    int // Current cache id for replace
}

// type misccache struct {
// 	cache     // Cache for other results
// 	id    int // Current cache id for misc computations
// }

// ************************************************************

// Hash value modifiers to distinguish between entries in misccache
// const cacheid_CONSTRAIN int = 0x0
// const cacheid_RESTRICT int = 0x1
// const cacheid_SATCOU int = 0x2
// const cacheid_SATCOULN int = 0x3
// const cacheid_PATHCOU int = 0x4

// Hash value modifiers for replace/compose
const cacheid_REPLACE int = 0x0

// const cacheid_COMPOSE int = 0x1
// const cacheid_VECCOMPOSE int = 0x2

// Hash value modifiers for quantification
const cacheid_EXIST int = 0x0
const cacheid_APPEX int = 0x3

// const cacheid_FORALL int = 0x1
// const cacheid_UNIQUE int = 0x2
// const cacheid_APPAL int = 0x4
// const cacheid_APPUN int = 0x5

// ************************************************************

// Basic functions shared by all caches

func (bc *cache) cacheinit(size int) {
	// we never check if the creation of the slice panic because of lack of memory
	size = bdd_prime_gte(size)
	bc.table = make([]cacheData, size)
	bc.cachereset()
}

func (bc *cache) cacheresize(size int) {
	// OPTIM: reuse the existing slice and append to it, or take a subslice if
	// we shrink the cache; not sure if it is possible
	if bc.cacheratio > 0 {
		bc.cacheinit(size / bc.cacheratio)
		return
	}
	bc.cachereset()
}

func (bc *cache) cachereset() {
	for k := range bc.table {
		bc.table[k].a = -1
	}
}

// *************************************************************************
// Setup and shutdown

func (b *buddy) cacheinit(cachesize int) {
	b.quantset = make([]int32, 0)
	if cachesize <= 0 {
		cachesize = len(b.nodes)/5 + 1
	}
	cachesize = bdd_prime_gte(cachesize)
	b.applycache = &applycache{}
	b.applycache.cacheinit(cachesize)
	b.itecache = &itecache{}
	b.itecache.cacheinit(cachesize)
	b.quantcache = &quantcache{}
	b.quantcache.cacheinit(cachesize)
	b.appexcache = &appexcache{}
	b.appexcache.cacheinit(cachesize)
	// b.misccache = &misccache{}
	// b.misccache.cacheinit(cachesize)
	b.replacecache = &replacecache{}
	b.replacecache.cacheinit(cachesize)
}

func (b *buddy) cachereset() {
	b.applycache.cachereset()
	b.itecache.cachereset()
	b.quantcache.cachereset()
	b.appexcache.cachereset()
	// b.misccache.cachereset()
	b.replacecache.cachereset()
}

func (b *buddy) cacheresize() {
	b.applycache.cacheresize(len(b.nodes))
	b.itecache.cacheresize(len(b.nodes))
	b.quantcache.cacheresize(len(b.nodes))
	b.appexcache.cacheresize(len(b.nodes))
	// b.misccache.cacheresize(len(b.nodes))
	b.replacecache.cacheresize(len(b.nodes))
}

// *************************************************************************

// SetCacheratio sets the cache ratio for the operator caches.
//
// The ratio between the number of nodes in the BDD table and the number of
// entries in the operator cachetables is called the cache ratio. So a cache
// ratio of say, four, allocates one cache entry for each four unique node
// entries. This value can be set to any positive value. When this is done the
// caches are resized instantly to fit the new ratio. The default is a fixed
// cache size determined at initialization time.
func (b *buddy) SetCacheratio(r int) error {
	if r <= 0 {
		b.seterror("Negative ratio (%d) in call to SetCacheratio", r)
		return b.error
	}
	if len(b.nodes) == 0 {
		return nil
	}
	b.applycache.cacheratio = r
	b.itecache.cacheratio = r
	b.quantcache.cacheratio = r
	b.appexcache.cacheratio = r
	// b.misccache.cacheratio = r
	b.replacecache.cacheratio = r
	b.cacheresize()
	return nil
}

// ************************************************************
//
// Quantification Cache
//

// quantset2cache takes a variable list, similar to the ones generated with
// Makeset, and set the variables in the quantification cache.
func (b *buddy) quantset2cache(n int) error {
	if n < 2 {
		b.seterror("Illegal variable (%d) in varset to cache", n)
		return b.error
	}
	b.quantsetID++
	if b.quantsetID == math.MaxInt32 {
		b.quantset = make([]int32, b.varnum)
		b.quantsetID = 1
	}
	for i := n; i > 1; i = b.nodes[i].high {
		b.quantset[b.nodes[i].level] = b.quantsetID
		b.quantlast = b.nodes[i].level
	}
	return nil
}

// ************************************************************

//
// Prints information about the cache performance. The information contains the
// number of accesses to the unique node table, the number of times a node was
// (not) found there and how many times a hash chain had to traversed. Hit and
// miss count is also given for the operator caches.

func (c cacheStat) String() string {
	res := fmt.Sprintf("Unique Access:  %d\n", c.uniqueAccess)
	res += fmt.Sprintf("Unique Chain:   %d\n", c.uniqueChain)
	res += fmt.Sprintf("Unique Hit:     %d\n", c.uniqueHit)
	res += fmt.Sprintf("Unique Miss:    %d\n", c.uniqueMiss)
	res += fmt.Sprintf("Operator Hits:  %d\n", c.opHit)
	res += fmt.Sprintf("Operator Miss:  %d", c.opMiss)
	return res
}
