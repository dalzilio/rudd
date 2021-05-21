// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd_test

import (
	"fmt"
	"log"

	"github.com/dalzilio/rudd"
)

// This example shows the basic usage of the package: create a BDD, compute some
// expressions and output the result.
func Example_basic() {
	// Create a new BDD with 6 variables, 10 000 nodes and a cache size of 5 000
	// (initially), with an implementation based on the BuDDY approach.
	bdd, _ := rudd.New(6, rudd.Nodesize(10000), rudd.Cachesize(3000))
	// n1 is a set comprising the three variables {x2, x3, x5}. It can also be
	// interpreted as the Boolean expression: x2 & x3 & x5
	n1 := bdd.Makeset([]int{2, 3, 5})
	// n2 == x1 | !x3 | x4
	n2 := bdd.Or(bdd.Ithvar(1), bdd.NIthvar(3), bdd.Ithvar(4))
	// n3 == ∃ x2,x3,x5 . (n2 & x3)
	n3 := bdd.AndExist(n1, n2, bdd.Ithvar(3))
	// You can print the result or export a BDD in Graphviz's DOT format
	log.Print("\n" + bdd.Stats())
	fmt.Printf("Number of sat. assignments is %s\n", bdd.Satcount(n3))
	// Output:
	// Number of sat. assignments is 48
}

// The following is an example of a callback handler, used in a call to Allsat,
// that counts the number of possible assignments (such that we do not count
// don't care twice).
func Example_allsat() {
	bdd, _ := rudd.New(5)
	// n == ∃ x2,x3 . (x1 | !x3 | x4) & x3
	n := bdd.AndExist(bdd.Makeset([]int{2, 3}),
		bdd.Or(bdd.Ithvar(1), bdd.NIthvar(3), bdd.Ithvar(4)),
		bdd.Ithvar(3))
	acc := new(int)
	bdd.Allsat(func(varset []int) error {
		*acc++
		return nil
	}, n)
	fmt.Printf("Number of sat. assignments (without don't care) is %d", *acc)
	// Output:
	// Number of sat. assignments (without don't care) is 2

}

// The following is an example of a callback handler, used in a call to
// Allnodes, that counts the number of active nodes in the whole BDD.
func Example_allnodes() {
	bdd, _ := rudd.New(5)
	n := bdd.AndExist(bdd.Makeset([]int{2, 3}),
		bdd.Or(bdd.Ithvar(1), bdd.NIthvar(3), bdd.Ithvar(4)),
		bdd.Ithvar(3))
	acc := new(int)
	count := func(id, level, low, high int) error {
		*acc++
		return nil
	}
	bdd.Allnodes(count)
	fmt.Printf("Number of active nodes in BDD is %d\n", *acc)
	*acc = 0
	bdd.Allnodes(count, n)
	fmt.Printf("Number of active nodes in node is %d", *acc)
	// Output:
	// Number of active nodes in BDD is 16
	// Number of active nodes in node is 2
}
