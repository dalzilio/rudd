// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd

import (
	"fmt"
	"math"
)

var _REPLACEID = 1

// Replacer is the type of association lists used to replace variables in a BDD
// node.
type Replacer interface {
	Replace(int32) (int32, bool)
	Id() int
}

type replacer struct {
	id    int     // unique identifier used for caching intermediate results
	image []int32 // map the level of old variables to the level of new variables
	last  int32   // last index in the Replacer, to speed up computations
}

func (r *replacer) String() string {
	res := fmt.Sprintf("replacer(last: %d)[", r.last)
	first := true
	for k, v := range r.image {
		if k != int(v) {
			if !first {
				res += ", "
			}
			first = false
			res += fmt.Sprintf("%d<-%d", k, v)
		}
	}
	return res + "]"
}

func (r *replacer) Replace(level int32) (int32, bool) {
	if level > r.last {
		return level, false
	}
	return r.image[level], true
}

func (r *replacer) Id() int {
	return r.id
}

// NewReplacer returns a Replacer for substituting variable oldvars[k] with
// newvars[k]. We return an error if the two slices do not have the same length
// or if we find the same index twice in either of them. All values must be in
// [0..Varnum).
func (b Set) NewReplacer(oldvars []int, newvars []int) (Replacer, error) {
	res := &replacer{}
	if len(oldvars) != len(newvars) {
		return nil, fmt.Errorf("unmatched length of slices")
	}
	if _REPLACEID == (math.MaxInt32 >> 2) {
		return nil, fmt.Errorf("too many replacers created")
	}
	res.id = (_REPLACEID << 2) | cacheid_REPLACE
	_REPLACEID++
	varnum := b.Varnum()
	support := make([]bool, varnum)
	res.image = make([]int32, varnum)
	for k := range res.image {
		res.image[k] = int32(k)
	}
	for k, v := range oldvars {
		if support[v] {
			return nil, fmt.Errorf("duplicate variable (%d) in oldvars", v)
		}
		if v >= varnum {
			return nil, fmt.Errorf("invalid variable in oldvars (%d)", v)
		}
		if newvars[k] >= varnum {
			return nil, fmt.Errorf("invalid variable in newvars (%d)", v)
		}
		support[v] = true
		res.image[v] = int32(newvars[k])
		if int32(v) > res.last {
			res.last = int32(v)
		}
	}
	for _, v := range newvars {
		if int(res.image[v]) != v {
			return nil, fmt.Errorf("variable in newvars (%d) also occur in oldvars", v)
		}
	}
	return res, nil
}

// ************************************************************

// Replace takes a Replacer and computes the result of n after replacing old
// variables with new ones. See type Replacer.
func (b *buddy) Replace(n Node, r Replacer) Node {
	if b.checkptr(n) != nil {
		return b.seterror("wrong operand in call to Replace (%d)", *n)
	}
	b.initref()
	b.pushref(*n)
	b.replacecache.id = r.Id()
	res := b.retnode(b.replace(*n, r))
	b.popref(1)
	return res
}

func (b *buddy) replace(n int, r Replacer) int {
	image, ok := r.Replace(b.nodes[n].level)
	if !ok {
		return n
	}
	if res := b.matchreplace(n); res >= 0 {
		return res
	}
	low := b.pushref(b.replace(b.nodes[n].low, r))
	high := b.pushref(b.replace(b.nodes[n].high, r))
	res := b.correctify(image, low, high)
	b.popref(2)
	return b.setreplace(n, res)
}

func (b *buddy) correctify(level int32, low, high int) int {
	/* FIXME: we do not use the cache here */
	if (level < b.nodes[low].level) && (level < b.nodes[high].level) {
		return b.makenode(level, low, high)
	}

	if (level == b.nodes[low].level) || (level == b.nodes[high].level) {
		b.seterror("error in replace level (%d) == low (%d:%d) or high (%d:%d)", level, low, b.nodes[low].level, high, b.nodes[high].level)
		return -1
	}

	if b.nodes[low].level == b.nodes[high].level {
		left := b.pushref(b.correctify(level, b.nodes[low].low, b.nodes[high].low))
		right := b.pushref(b.correctify(level, b.nodes[low].high, b.nodes[high].high))
		res := b.makenode(b.nodes[low].level, left, right)
		b.popref(2)
		return res
	}

	if b.nodes[low].level < b.nodes[high].level {
		left := b.pushref(b.correctify(level, b.nodes[low].low, high))
		right := b.pushref(b.correctify(level, b.nodes[low].high, high))
		res := b.makenode(b.nodes[low].level, left, right)
		b.popref(2)
		return res
	}

	left := b.pushref(b.correctify(level, low, b.nodes[high].low))
	right := b.pushref(b.correctify(level, low, b.nodes[high].high))
	res := b.makenode(b.nodes[high].level, left, right)
	b.popref(2)
	return res
}
