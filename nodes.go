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

type bddNode struct {
	refcou int32 // Count the number of external references
	level  int32 // Order of the variable in the BDD
	low    int   // Reference to the false branch
	high   int   // Reference to the true branch
	hash   int   // Index where to (possibly) find node with this hash value
	next   int   // Next index to check in case of a collision, 0 if last
}

// ************************************************************

func (b *BDD) ismarked(n int) bool {
	return (b.nodes[n].level & 0x200000) != 0
}

func (b *BDD) marknode(n int) {
	b.nodes[n].level = b.nodes[n].level | 0x200000
}

func (b *BDD) unmarknode(n int) {
	b.nodes[n].level = b.nodes[n].level & 0x1FFFFF
}
