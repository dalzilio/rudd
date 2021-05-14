// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd

import "log"

// SetVarnum sets the number of BDD variables. This function is used to define
// the number of variables used in the BDD package. It may be called more than
// one time, but only to increase the number of variables.
func (b *buddy) SetVarnum(num int) error {
	oldvarnum := b.varnum
	inum := int32(num)
	if (inum < 1) || (inum > _MAXVAR) {
		b.seterror("Bad number of variable (%d) in setvarnum", inum)
		return b.error
	}
	if inum < b.varnum {
		b.seterror("Trying to decrease the number of variables in SetVarnum (from %d to %d)", b.varnum, inum)
		return b.error
	}
	if inum == b.varnum {
		return b.error
	}

	// We create new slices for the fields related to the list of variables:
	// varset, level2var, var2level.
	tmpvarset := b.varset
	b.varset = make([][2]int, inum)
	copy(b.varset, tmpvarset)

	// Constants always have the highest level.
	b.nodes[0].level = inum
	b.nodes[1].level = inum

	// We also initialize the refstack.
	b.refstack = make([]int, 0, 2*inum+4)
	b.initref()
	for ; b.varnum < inum; b.varnum++ {
		v0 := b.makenode(b.varnum, 0, 1)
		if v0 < 0 {
			b.varnum = oldvarnum
			b.seterror("Cannot allocate new variable %d in SetVarnum; %s", b.varnum, b.error)
			return b.error
		}
		b.pushref(v0)
		v1 := b.makenode(b.varnum, 1, 0)
		if v1 < 0 {
			b.varnum = oldvarnum
			b.seterror("Cannot allocate new variable %d in SetVarnum; %s", b.varnum, b.error)
			return b.error

		}
		b.popref(1)
		b.varset[b.varnum] = [2]int{v0, v1}
		b.nodes[b.varset[b.varnum][0]].refcou = _MAXREFCOUNT
		b.nodes[b.varset[b.varnum][1]].refcou = _MAXREFCOUNT
	}

	// We also need to resize the quantification cache
	b.quantset = make([]int32, b.varnum)
	b.quantsetID = 0

	if _LOGLEVEL > 0 {
		log.Printf("set varnum to %d\n", b.varnum)
	}
	return nil
}

// *************************************************************************

// ExtVarnum extends the current number of allocated BDD variables with num
// extra variables
func (b *buddy) ExtVarnum(num int) error {
	if (num < 0) || (num > 0x3FFFFFFF) {
		b.seterror("Bad choice of value (%d) when extending varnum in ExtVarnum", num)
		return b.error
	}
	return b.SetVarnum(int(b.varnum) + num)
}
