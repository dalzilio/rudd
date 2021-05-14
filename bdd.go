// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd

import "math/big"

// Set encapsulates the access to a BDD implementation and provides additionnal
// functions to ease the display and computation of arbitrary Boolean
// expressions.
type Set struct {
	// we embedd the BDD interface in order to implement methods with a Set
	// receiver
	BDD
}

// BDD is an interface implementing the basic operations over Binary Decision
// Diagrams.
type BDD interface {
	// Error returns the error status of the BDD. We return an empty string if
	// there are no errors.
	Error() string

	// SetVarnum sets the number of BDD variables. It may be called more than
	// once, but only to increase the number of variables.
	SetVarnum(num int) error

	// Varnum returns the number of defined variables.
	Varnum() int

	// True returns the Node for the constant true.
	True() Node

	// False returns the Node for the constant false.
	False() Node

	// From returns a (constant) Node from a boolean value.
	From(v bool) Node

	// Ithvar returns a BDD representing the i'th variable on success. The
	// requested variable must be in the range [0..Varnum).
	Ithvar(i int) Node

	// NIthvar returns a bdd representing the negation of the i'th variable on
	// success. See *ithvar* for further info.
	NIthvar(i int) Node

	// Low returns the false branch of a BDD or nil if there is an error.
	Low(n Node) Node

	// High returns the true branch of a BDD.
	High(n Node) Node

	// Makeset returns a node corresponding to the conjunction (the cube) of all
	// the variables in varset, in their positive form. It is such that
	// scanset(Makeset(a)) == a. It returns nil if one of the variables is
	// outside the scope of the BDD (see documentation for function *Ithvar*).
	Makeset(varset []int) Node

	// Scanset returns the set of variables found when following the high branch
	// of node n. This is the dual of function Makeset. The result may be nil if
	// there is an error and it is an empty slice if the set is empty.
	Scanset(n Node) []int

	// Not returns the negation (!n) of expression n.
	Not(n Node) Node

	// Apply performs all of the basic binary operations on BDD nodes, such as
	// AND, OR etc.
	Apply(left Node, right Node, op Operator) Node

	// Ite, short for if-then-else operator, computes the BDD for the expression
	// [(f &  g) | (!f & h)] more efficiently than doing the three operations
	// separately.
	Ite(f, g, h Node) Node

	// Exist returns the existential quantification of n for the variables in
	// varset, where varset is a node built with a method such as Makeset.
	Exist(n, varset Node) Node

	// AppEx applies the binary operator *op* on the two operands left and right
	// then performs an existential quantification over the variables in varset,
	// where varset is a node computed with an operation such as Makeset.
	AppEx(left Node, right Node, op Operator, varset Node) Node

	// Replace takes a renamer and computes the result of n after replacing old
	// variables with new ones. See type Renamer.
	Replace(n Node, r Replacer) Node

	// Satcount computes the number of satisfying variable assignments for the
	// function denoted by n. We return a result using arbitrary-precision
	// arithmetic to avoid possible overflows. The result is zero (and we set
	// the error flag of b) if there is an error.
	Satcount(n Node) *big.Int

	// Allsat Iterates through all legal variable assignments for n and calls
	// the function f on each of them. We pass an int slice of length varnum to
	// f where each entry is either 0 if the variable is false, 1 if it is true,
	// and -1 if it is a don't care. We stop and return an error if f returns an
	// error at some point.
	Allsat(n Node, f func([]int) error) error

	// Allnodes is similar to Allsat but iterates over all the nodes accessible
	// from one of the parameters in n (or all the active nodes if n is absent).
	// Function f takes the id, level, and id's of the low and high successors
	// of each node. The two constant nodes (True and False) have always the id
	// 1 and 0 respectively.
	Allnodes(f func(id, level, low, high int) error, n ...Node) error

	// // AddRef increases the reference count on node n and returns n so that
	// // calls can be easily chained together. A call to AddRef can never raise an
	// // error, even if we access an unused node or a value outside the range of
	// // the BDD.
	// AddRef(n Node) Node

	// // DelRef decreases the reference count on a node and returns n so that
	// // calls can be easily chained together. A call to DelRef can never raise an
	// // error, even if we access an unused node or a value outside the range of
	// // the BDD.
	// DelRef(n Node) Node

	// // GC explitly starts garbage collection of unused nodes.
	// GC()

	// Stats returns information about the BDD
	Stats() string
}

// ************************************************************

// Node is a reference to an element of a BDD. It represents the atomic unit of
// interactions and computations within a BDD.
type Node *int

// ************************************************************

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

// *************************************************************************
