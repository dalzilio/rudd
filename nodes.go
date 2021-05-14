// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

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

func (b *buddy) ismarked(n int) bool {
	return (b.nodes[n].level & 0x200000) != 0
}

func (b *buddy) marknode(n int) {
	b.nodes[n].level = b.nodes[n].level | 0x200000
}

func (b *buddy) unmarknode(n int) {
	b.nodes[n].level = b.nodes[n].level & 0x1FFFFF
}
