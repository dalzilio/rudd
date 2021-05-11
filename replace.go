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

var _RENAMERID = 0

// Renamer is an object used to replace variables in a BDD node.
type Renamer struct {
	id    int     // unique identifier used for caching intermediate results
	image []int32 // map the level of old variables to the level of new variables
	last  int32   // last index in the renamer, to speed up computations
}

func (r *Renamer) String() string {
	res := fmt.Sprintf("renamer(last: %d)[", r.last)
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

// NewRenamer returns a renamer for substituting variable oldvars[k] with
// newvars[k]. We return an error if the two slices do not have the same length
// or if we find the same index twice in either of them. All values must be
// valid variable levels in the BDD.
func (b *BDD) NewRenamer(oldvars []int, newvars []int) (*Renamer, error) {
	res := &Renamer{}
	if len(oldvars) != len(newvars) {
		return nil, fmt.Errorf("unmatched length of slices")
	}
	if _RENAMERID == (math.MaxInt32 >> 2) {
		return nil, fmt.Errorf("too many renamers created")
	}
	res.id = _RENAMERID
	_RENAMERID++
	support := make([]bool, b.varnum)
	res.image = make([]int32, b.varnum)
	for k := range res.image {
		res.image[k] = int32(k)
	}
	for k, v := range oldvars {
		if support[b.level2var[v]] {
			return nil, fmt.Errorf("duplicate variable (%d) in oldvars", v)
		}
		if v >= int(b.varnum) {
			return nil, fmt.Errorf("invalid variable in oldvars (%d)", v)
		}
		if newvars[k] >= int(b.varnum) {
			return nil, fmt.Errorf("invalid variable in newvars (%d)", v)
		}
		support[b.level2var[v]] = true
		res.image[b.level2var[v]] = int32(b.level2var[newvars[k]])
		if b.level2var[v] > res.last {
			res.last = b.level2var[v]
		}
	}
	for _, v := range newvars {
		if b.level2var[res.image[v]] != b.level2var[v] {
			return nil, fmt.Errorf("variable in newvars (%d) also occur in oldvars", v)
		}
	}
	return res, nil
}

// ************************************************************

// Replace takes a renamer and computes the result of n after replacing old
// variables with new ones. See type Renamer.
func (b *BDD) Replace(n Node, r *Renamer) Node {
	if b.checkptr(n) != nil {
		b.seterror("wrong operand in call to Replace (%d)", *n)
		return bdderror
	}
	b.initref()
	b.replacecache.id = r.id<<2 | cacheid_REPLACE
	return b.retnode(b.replace(*n, r))
}

func (b *BDD) replace(n int, r *Renamer) int {

	if n < 2 || b.nodes[n].level > r.last {
		return n
	}

	if res := b.matchreplace(n); res >= 0 {
		if _DEBUG {
			b.cacheStat.opHit++
		}
		return res
	}
	if _DEBUG {
		b.cacheStat.opMiss++
	}

	low := b.pushref(b.replace(b.nodes[n].low, r))
	high := b.pushref(b.replace(b.nodes[n].high, r))
	res := b.correctify(r.image[b.nodes[n].level], low, high)
	b.popref(2)
	return b.setreplace(n, res)
}

func (b *BDD) correctify(level int32, low, high int) int {
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
