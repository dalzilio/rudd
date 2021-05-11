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
	"log"
	"math/big"
)

type Operator int

// Operator describe the potential (binary) operations available on an Apply.
const (
	OPand       Operator = iota // Boolean conjunction
	OPxor                       // Exclusive or
	OPor                        // Disjunction
	OPnand                      // Negation of and
	OPnor                       // Negation of or
	OPimp                       // Implication
	OPbiimp                     // Equivalence
	OPdiff                      // Difference
	OPless                      // Set difference
	OPinvimp                    // Reverse implication
	op_not                      // Negation. Should not be used in apply, but used in caches
	op_simplify                 // same
)

var opnames = [12]string{
	OPand:       "and",
	OPxor:       "xor",
	OPor:        "or",
	OPnand:      "nand",
	OPnor:       "nor",
	OPimp:       "imp",
	OPbiimp:     "biimp",
	OPdiff:      "diff",
	OPless:      "less",
	OPinvimp:    "invimp",
	op_not:      "not",
	op_simplify: "simplify",
}

func (op Operator) String() string {
	return opnames[op]
}

var opres = [12][2][2]int{
	//                      00    01               10    11
	OPand:    {0: [2]int{0: 0, 1: 0}, 1: [2]int{0: 0, 1: 1}}, // 0001
	OPxor:    {0: [2]int{0: 0, 1: 1}, 1: [2]int{0: 1, 1: 0}}, // 0110
	OPor:     {0: [2]int{0: 0, 1: 1}, 1: [2]int{0: 1, 1: 1}}, // 0111
	OPnand:   {0: [2]int{0: 1, 1: 1}, 1: [2]int{0: 1, 1: 0}}, // 1110
	OPnor:    {0: [2]int{0: 1, 1: 0}, 1: [2]int{0: 0, 1: 0}}, // 1000
	OPimp:    {0: [2]int{0: 1, 1: 1}, 1: [2]int{0: 0, 1: 1}}, // 1101
	OPbiimp:  {0: [2]int{0: 1, 1: 0}, 1: [2]int{0: 0, 1: 1}}, // 1001
	OPdiff:   {0: [2]int{0: 0, 1: 0}, 1: [2]int{0: 1, 1: 0}}, // 0010
	OPless:   {0: [2]int{0: 0, 1: 1}, 1: [2]int{0: 0, 1: 0}}, // 0100
	OPinvimp: {0: [2]int{0: 1, 1: 0}, 1: [2]int{0: 1, 1: 1}}, // 1011
}

// *************************************************************************

// Not returns the negation of the expression corresponding to node n. It
// negates a BDD by exchanging all references to the zero-terminal with
// references to the one-terminal and vice versa.
func (b *BDD) Not(n Node) Node {
	if b.checkptr(n) != nil {
		return b.seterror("Wrong operand in call to Not (%d)", *n)
	}
	b.initref()
	b.pushref(*n)
	res := b.not(*n)
	b.popref(1)
	return b.retnode(res)
}

func (b *BDD) not(n int) int {
	if n == 0 {
		return 1
	}
	if n == 1 {
		return 0
	}
	// The hash for a not operation is simply n
	if res := b.matchnot(n); res >= 0 {
		if _DEBUG {
			b.cacheStat.opHit++
		}
		return res
	}
	if _DEBUG {
		b.cacheStat.opMiss++
	}
	low := b.pushref(b.not(b.nodes[n].low))
	high := b.pushref(b.not(b.nodes[n].high))
	res := b.makenode(b.nodes[n].level, low, high)
	b.popref(2)
	return b.setnot(n, res)
}

// *************************************************************************

// Apply performs all of the basic bdd operations with two operands, such as
// AND, OR etc. Left and right are the operand and opr is the requested
// operation and must be one of the following:
//
//  Identifier    Description			 Truth table
//
//  OPand		  logical and    		 [0,0,0,1]
//  OPxor		  logical xor     		 [0,1,1,0]
//	OPor		  logical or   			 [0,1,1,1]
// 	OPnand 		  logical not-and		 [1,1,1,0]
// 	OPnor		  logical not-or    	 [1,0,0,0]
// 	OPimp		  implication 			 [1,1,0,1]
// 	OPbiimp		  equivalence			 [1,0,0,1]
// 	OPdiff		  set difference 		 [0,0,1,0]
// 	OPless   	  less than				 [0,1,0,0]
//  OPinvimp	  reverse implication 	 [1,0,1,1]
func (b *BDD) Apply(left Node, right Node, op Operator) Node {
	if b.checkptr(left) != nil {
		return b.seterror("Wrong operand in call to Apply %s(left: %d, right: ...)", op, *left)
	}
	if b.checkptr(right) != nil {
		return b.seterror("Wrong operand in call to Apply %s(left: ..., right: %d)", op, *right)
	}
	b.applycache.op = op
	b.initref()
	b.pushref(*left)
	b.pushref(*right)
	res := b.apply(*left, *right)
	b.popref(2)
	return b.retnode(res)
}

func (b *BDD) apply(left int, right int) int {
	switch b.applycache.op {
	case OPand:
		if left == right {
			return left
		}
		if (left == 0) || (right == 0) {
			return 0
		}
		if left == 1 {
			return right
		}
		if right == 1 {
			return left
		}
	case OPor:
		if left == right {
			return left
		}
		if (left == 1) || (right == 1) {
			return 1
		}
		if left == 0 {
			return right
		}
		if right == 0 {
			return left
		}
	case OPxor:
		if left == right {
			return 0
		}
		if left == 0 {
			return right
		}
		if right == 0 {
			return left
		}
	case OPnand:
		if (left == 0) || (right == 0) {
			return 1
		}
	case OPnor:
		if (left == 1) || (right == 1) {
			return 0
		}
	case OPimp:
		if left == 0 {
			return 1
		}
		if left == 1 {
			return right
		}
		if right == 1 {
			return 1
		}
		if left == right {
			return 1
		}
	case OPbiimp:
		if left == right {
			return 1
		}
		if left == 1 {
			return right
		}
		if right == 1 {
			return left
		}
	case OPdiff:
		if left == right {
			return 0
		}
		if right == 1 {
			return 0
		}
		if left == 0 {
			return right
		}
	case OPless:
		if (left == right) || (left == 1) {
			return 0
		}
		if left == 0 {
			return right
		}
	case OPinvimp:
		if right == 0 {
			return 1
		}
		if right == 1 {
			return left
		}
		if left == 1 {
			return 1
		}
		if left == right {
			return 1
		}
	default:
		// unary operations, OPnot and OPsimplify, should not be used in apply
		b.seterror("Unauthorized operation (%s) in apply", b.applycache.op)
		return -1
	}

	// we check for errors
	if left < 0 || right < 0 {
		if _DEBUG {
			log.Panicf("panic in apply(%d,%d,%s)\n", left, right, b.applycache.op)
		}
		return -1
	}

	// we deal with the other cases where the two operands are constants
	if (left < 2) && (right < 2) {
		return opres[b.applycache.op][left][right]
	}

	// otherwise we check the cache
	if res := b.matchapply(left, right); res >= 0 {
		if _DEBUG {
			b.cacheStat.opHit++
		}
		return res
	}
	// if we are unfortunate we continue recursively
	if _DEBUG {
		b.cacheStat.opMiss++
	}
	leftlvl := b.nodes[left].level
	rightlvl := b.nodes[right].level
	var res int
	if leftlvl == rightlvl {
		low := b.pushref(b.apply(b.nodes[left].low, b.nodes[right].low))
		high := b.pushref(b.apply(b.nodes[left].high, b.nodes[right].high))
		res = b.makenode(leftlvl, low, high)
	} else {
		if leftlvl < rightlvl {
			low := b.pushref(b.apply(b.nodes[left].low, right))
			high := b.pushref(b.apply(b.nodes[left].high, right))
			res = b.makenode(leftlvl, low, high)
		} else {
			low := b.pushref(b.apply(left, b.nodes[right].low))
			high := b.pushref(b.apply(left, b.nodes[right].high))
			res = b.makenode(rightlvl, low, high)
		}
	}
	b.popref(2)
	return b.setapply(left, right, res)
}

// *************************************************************************

// Ite, short for if-then-else operator, computes the BDD for the expression [(f
// /\ g) \/ (not f /\ h)] more efficiently than doing the three operations
// separately.
func (b *BDD) Ite(f, g, h Node) Node {
	if b.checkptr(f) != nil {
		return b.seterror("Wrong operand in call to Ite (f: %d)", *f)
	}
	if b.checkptr(g) != nil {
		return b.seterror("Wrong operand in call to Ite (g: %d)", *g)
	}
	if b.checkptr(h) != nil {
		return b.seterror("Wrong operand in call to Ite (h: %d)", *h)
	}
	b.initref()
	b.pushref(*f)
	b.pushref(*g)
	b.pushref(*h)
	res := b.ite(*f, *g, *h)
	b.popref(3)
	return b.retnode(res)
}

// ite_low returns p if p is strictly higher than q or r, otherwise it returns
// p.low. This is used in function ite to know which node to follow: we always
// follow the smallest(s) nodes.
func (b *BDD) ite_low(p, q, r int32, n int) int {
	if (p > q) || (p > r) {
		return n
	}
	return b.nodes[n].low
}

func (b *BDD) ite_high(p, q, r int32, n int) int {
	if (p > q) || (p > r) {
		return n
	}
	return b.nodes[n].high
}

// min3 returns the smallest value between p, q and r. This is used in function
// ite to compute the smallest level.
func min3(p, q, r int32) int32 {
	if p <= q {
		if p <= r { // p <= q && p <= r
			return p
		}
		return r // r < p <= q
	}
	if q <= r { // q < p && q <= r
		return q
	}
	return r // r < q < p
}

func (b *BDD) ite(f, g, h int) int {
	switch {
	case f == 1:
		return g
	case f == 0:
		return h
	case g == h:
		return g
	case (g == 1) && (h == 0):
		return f
	case (g == 0) && (h == 1):
		return b.not(f)
	}
	// we check for possible errors
	if f < 0 || g < 0 || h < 0 {
		b.seterror("unexpected error in ite")
		if _DEBUG {
			log.Panicf("panic in ite(%d,%d,%d)\n", f, g, h)
		}
		return -1
	}
	if res := b.matchite(f, g, h); res >= 0 {
		if _DEBUG {
			b.cacheStat.opHit++
		}
		return res
	}
	if _DEBUG {
		b.cacheStat.opMiss++
	}
	p := b.nodes[f].level
	q := b.nodes[g].level
	r := b.nodes[h].level
	low := b.pushref(b.ite(b.ite_low(p, q, r, f), b.ite_low(q, p, r, g), b.ite_low(r, p, q, h)))
	high := b.pushref(b.ite(b.ite_high(p, q, r, f), b.ite_high(q, p, r, g), b.ite_high(r, p, q, h)))
	res := b.makenode(min3(p, q, r), low, high)
	b.popref(2)
	return b.setite(f, g, h, res)
}

// *************************************************************************

// And returns the logical 'and' of a sequence of nodes.
func (b *BDD) And(n ...Node) Node {
	if len(n) == 1 {
		return n[0]
	}
	if len(n) == 0 {
		return bddone
	}
	return b.Apply(n[0], b.And(n[1:]...), OPand)
}

// Or returns the logical 'or' of a sequence of BDDs.
func (b *BDD) Or(n ...Node) Node {
	if len(n) == 1 {
		return n[0]
	}
	if len(n) == 0 {
		return bddzero
	}
	return b.Apply(n[0], b.Or(n[1:]...), OPor)
}

// Xor returns the logical 'xor' of two BDDs.
func (b *BDD) Xor(low, high Node) Node {
	return b.Apply(low, high, OPxor)
}

// Imp returns the logical 'implication' between two BDDs.
func (b *BDD) Imp(low, high Node) Node {
	return b.Apply(low, high, OPimp)
}

// Equiv returns the logical 'bi-implication' between two BDDs.
func (b *BDD) Equiv(low, high Node) Node {
	return b.Apply(low, high, OPbiimp)
}

// Equal tests equivalence between nodes.
func (b *BDD) Equal(low, high Node) bool {
	return *low == *high
}

// *************************************************************************

// Exist returns the existential quantification of n for the variables in
// varset, where varset is a node built with a method such as Makeset. We return
// bdderror and set the error flag in b if there is an error.
func (b *BDD) Exist(n, varset Node) Node {
	if b.checkptr(n) != nil {
		return b.seterror("Wrong node in call to Exist (n: %d)", *n)
	}
	if b.checkptr(varset) != nil {
		return b.seterror("Wrong varset in call to Exist (%d)", *varset)
	}
	if err := b.quantset2cache(*varset); err != nil {
		return bdderror
	}
	if *varset < 2 { // we have an empty set or a constant
		return n
	}

	b.quantcache.id = (*varset << 3) | cacheid_EXIST
	// FIXME: range, should check thet varset < 1 << (bits.UintSize - 3); but very unlikely
	b.applycache.op = OPor
	b.initref()
	b.pushref(*n)
	res := b.quant(*n)
	b.popref(1)
	return b.retnode(res)
}

func (b *BDD) quant(n int) int {
	if (n < 2) || (b.nodes[n].level > b.quantlast) {
		return n
	}
	// the hash for a quantification operation is simply n
	if res := b.matchquant(n); res >= 0 {
		if _DEBUG {
			b.cacheStat.opHit++
		}
		return res
	}
	if _DEBUG {
		b.cacheStat.opMiss++
	}
	low := b.pushref(b.quant(b.nodes[n].low))
	high := b.pushref(b.quant(b.nodes[n].high))
	var res int
	if b.quantset[b.nodes[n].level] == b.quantsetID {
		res = b.apply(low, high)
	} else {
		res = b.makenode(b.nodes[n].level, low, high)
	}
	b.popref(2)
	return b.setquant(n, res)
}

// *************************************************************************

// AppEx applies the binary operator *op* on the two operands left and right
// then performs an existential quantification over the variables in varset.
// This is done in a bottom up manner such that both the apply and
// quantification is done on the lower nodes before stepping up to the higher
// nodes. This makes AppEx much more efficient than an apply operation followed
// by a quantification. Note that, when *op* is a conjunction, this operation
// returns the relational product of two BDDs.
func (b *BDD) AppEx(left Node, right Node, op Operator, varset Node) Node {
	// FIXME: should check that op is a binary operation
	if b.checkptr(varset) != nil {
		return b.seterror("Wrong varset in call to AppEx (%d)", *varset)
	}
	if *varset < 2 { // we have an empty set
		return b.Apply(left, right, op)
	}
	if b.checkptr(left) != nil {
		return b.seterror("Wrong operand in call to AppEx %s(left: %d)", op, *left)
	}
	if b.checkptr(right) != nil {
		return b.seterror("Wrong operand in call to AppEx %s(right: %d)", op, *right)
	}
	if err := b.quantset2cache(*varset); err != nil {
		return bdderror
	}

	b.applycache.op = OPor
	b.appexcache.op = op
	b.appexcache.id = (*varset << 5) | (int(op) << 1) /* FIXME: range! */
	b.quantcache.id = (b.appexcache.id << 3) | cacheid_APPEX
	b.initref()
	b.pushref(*left)
	b.pushref(*right)
	b.pushref(*varset)
	res := b.appquant(*left, *right)
	b.popref(3)
	return b.retnode(res)
}

func (b *BDD) appquant(left, right int) int {
	switch b.appexcache.op {
	case OPand:
		if left == 0 || right == 0 {
			return 0
		}
		if left == right {
			return b.quant(left)
		}
		if left == 1 {
			return b.quant(right)
		}
		if right == 1 {
			return b.quant(left)
		}
	case OPor:
		if left == 1 || right == 1 {
			return 1
		}
		if left == right {
			return b.quant(left)
		}
		if left == 0 {
			return b.quant(right)
		}
		if right == 0 {
			return b.quant(left)
		}
	case OPxor:
		if left == right {
			return 0
		}
		if left == 0 {
			return b.quant(right)
		}
		if right == 0 {
			return b.quant(left)
		}
	case OPnand:
		if left == 0 || right == 0 {
			return 1
		}
	case OPnor:
		if left == 1 || right == 1 {
			return 0
		}
	default:
		// OPnot and OPsimplify should not be used in apply.
		//
		// FIXME: we are raising an error for other operations that would be OK.
		b.seterror("unauthorized operation (%s) in AppEx", b.applycache.op)
		return -1
	}

	// we check for errors
	if left < 0 || right < 0 {
		b.seterror("unexpected error in appquant")
		return -1
	}

	// we deal with the other cases when the two operands are constants
	if (left < 2) && (right < 2) {
		return opres[b.appexcache.op][left][right]
	}

	// and the case where we have no more variables to quantify
	if (b.nodes[left].level > b.quantlast) && (b.nodes[right].level > b.quantlast) {
		oldop := b.applycache.op
		b.applycache.op = b.appexcache.op
		res := b.apply(left, right)
		b.applycache.op = oldop
		return res
	}

	// next we check if the operation is already in our cache
	if res := b.matchappex(left, right); res >= 0 {
		if _DEBUG {
			b.cacheStat.opHit++
		}
		return res
	}
	if _DEBUG {
		b.cacheStat.opMiss++
	}
	leftlvl := b.nodes[left].level
	rightlvl := b.nodes[right].level
	var res int
	if leftlvl == rightlvl {
		low := b.pushref(b.appquant(b.nodes[left].low, b.nodes[right].low))
		high := b.pushref(b.appquant(b.nodes[left].high, b.nodes[right].high))
		if b.quantset[leftlvl] == b.quantsetID {
			res = b.apply(low, high)
		} else {
			res = b.makenode(leftlvl, low, high)
		}
	} else {
		if leftlvl < rightlvl {
			low := b.pushref(b.appquant(b.nodes[left].low, right))
			high := b.pushref(b.appquant(b.nodes[left].high, right))
			if b.quantset[leftlvl] == b.quantsetID {
				res = b.apply(low, high)
			} else {
				res = b.makenode(leftlvl, low, high)
			}
		} else {
			low := b.pushref(b.appquant(left, b.nodes[right].low))
			high := b.pushref(b.appquant(left, b.nodes[right].high))
			if b.quantset[rightlvl] == b.quantsetID {
				res = b.apply(low, high)
			} else {
				res = b.makenode(rightlvl, low, high)
			}
		}
	}
	b.popref(2)
	return b.setappex(left, right, res)
}

// *************************************************************************

// Satcount computes the number of satisfying variable assignments for the
// function denoted by n. We return a result using arbitrary-precision
// arithmetic to avoid possible overflows. The result is zero (and we set the
// error flag of b) if there is an error.
func (b *BDD) Satcount(n Node) *big.Int {
	res := big.NewInt(0)
	if b.checkptr(n) != nil {
		b.seterror("Wrong operand in call to Satcount (%d)", *n)
		return res
	}
	// We compute 2^level with a bit shift 1 << level
	res.SetBit(res, int(b.nodes[*n].level), 1)
	satc := make(map[int]*big.Int)
	return res.Mul(res, b.satcount(*n, satc))
}

func (b *BDD) satcount(n int, satc map[int]*big.Int) *big.Int {
	if n < 2 {
		return big.NewInt(int64(n))
	}
	// we use satc to memoize the value of satcount for each nodes
	res, ok := satc[n]
	if ok {
		return res
	}
	level := b.nodes[n].level
	low := b.nodes[n].low
	high := b.nodes[n].high

	res = big.NewInt(0)
	two := big.NewInt(0)
	two.SetBit(two, int(b.nodes[low].level-level-1), 1)
	res.Add(res, two.Mul(two, b.satcount(low, satc)))
	two = big.NewInt(0)
	two.SetBit(two, int(b.nodes[high].level-level-1), 1)
	res.Add(res, two.Mul(two, b.satcount(high, satc)))
	satc[n] = res
	return res
}

// *************************************************************************

// Allsat Iterates through all legal variable assignments for n and calls the
// function f on each of them. We pass an int slice of length varnum to f where
// ach entry is either  0 if the variable is false, 1 if it is true, and -1 if
// it is a don't care. We stop and return an error if f returns an error at some
// point.
//
// The following is an example of a callback handler that counts the number of
// possible assignments (such that we do not count don't care twice):
// 	   acc := new(int)
//     b.allsat(*n, prof, func(varset []int) error {
// 	     *acc++
// 	      return nil
//      })
func (b *BDD) Allsat(n Node, f func([]int) error) error {
	if b.checkptr(n) != nil {
		return fmt.Errorf("wrong node in call to Allsat (%d)", *n)
	}
	prof := make([]int, b.varnum)
	for k := range prof {
		prof[k] = -1
	}
	// the function does not create new nodes, so we do not need to take care of
	// possible resizing
	return b.allsat(*n, prof, f)
}

func (b *BDD) allsat(n int, prof []int, f func([]int) error) error {
	if n == 1 {
		return f(prof)
	}
	if n == 0 {
		return nil
	}

	if low := b.nodes[n].low; low != 0 {
		prof[b.level2var[b.nodes[n].level]] = 0
		for v := b.nodes[low].level - 1; v > b.nodes[n].level; v-- {
			prof[b.level2var[v]] = -1
		}
		if err := b.allsat(low, prof, f); err != nil {
			return nil
		}
	}

	if high := b.nodes[n].high; high != 0 {
		prof[b.level2var[b.nodes[n].level]] = 1
		for v := b.nodes[high].level - 1; v > b.nodes[n].level; v-- {
			prof[b.level2var[v]] = -1
		}
		if err := b.allsat(high, prof, f); err != nil {
			return nil
		}
	}
	return nil
}
