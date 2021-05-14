// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd

import "log"

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

// // *************************************************************************

// // ExtVarnum extends the current number of allocated BDD variables with num
// // extra variables
// func (b *buddy) ExtVarnum(num int) error {
// 	if (num < 0) || (num > 0x3FFFFFFF) {
// 		b.seterror("Bad choice of value (%d) when extending varnum in ExtVarnum", num)
// 		return b.error
// 	}
// 	return b.SetVarnum(int(b.varnum) + num)
// }
