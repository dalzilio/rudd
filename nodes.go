// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd

type buddyNode struct {
	refcou int32 // Count the number of external references
	level  int32 // Order of the variable in the BDD
	low    int   // Reference to the false branch
	high   int   // Reference to the true branch
	hash   int   // Index where to (possibly) find node with this hash value
	next   int   // Next index to check in case of a collision, 0 if last
}

// ************************************************************

// inode returns a Node for known nodes, such as variables, that do not need to
// increase their reference count.
func inode(n int) Node {
	x := n
	return &x
}

var bddone Node = inode(1)

var bddzero Node = inode(0)

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
