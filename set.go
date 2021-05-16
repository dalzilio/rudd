// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd

// And returns the logical 'and' of a sequence of nodes.
func (b Set) And(n ...Node) Node {
	if len(n) == 1 {
		return n[0]
	}
	if len(n) == 0 {
		return bddone
	}
	return b.Apply(n[0], b.And(n[1:]...), OPand)
}

// Or returns the logical 'or' of a sequence of BDDs.
func (b Set) Or(n ...Node) Node {
	if len(n) == 1 {
		return n[0]
	}
	if len(n) == 0 {
		return bddzero
	}
	return b.Apply(n[0], b.Or(n[1:]...), OPor)
}

// Imp returns the logical 'implication' between two BDDs.
func (b Set) Imp(n1, n2 Node) Node {
	return b.Apply(n1, n2, OPimp)
}

// Equiv returns the logical 'bi-implication' between two BDDs.
func (b Set) Equiv(n1, n2 Node) Node {
	return b.Apply(n1, n2, OPbiimp)
}

// Equal tests equivalence between nodes.
func (b Set) Equal(low, high Node) bool {
	if low == high {
		return true
	}
	if low == nil || high == nil {
		return false
	}
	return *low == *high
}

// AndExists returns the "relational composition" of two nodes with respect to
// varset, meaning the result of (Exists varset . n1 & n2).
func (b Set) AndExist(varset, n1, n2 Node) Node {
	return b.AppEx(n1, n2, OPand, varset)
}

// True returns the constant true BDD
func (b Set) True() Node {
	return bddone
}

// False returns the constant false BDD
func (b Set) False() Node {
	return bddzero
}

// From returns a (constant) Node from a boolean value.
func (b Set) From(v bool) Node {
	if v {
		return bddone
	}
	return bddzero
}
