// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestIte(t *testing.T) {
	bdd, _ := New(4, Nodesize(5000), Cachesize(1000))
	n1 := bdd.Makeset([]int{0, 2, 3})
	n2 := bdd.Makeset([]int{0, 3})
	actual := bdd.Equiv(bdd.Ite(n1, n2, bdd.Not(n2)), bdd.Or(bdd.And(n1, n2), bdd.And(bdd.Not(n1), bdd.Not(n2))))
	if actual != bdd.True() {
		t.Error("ite(f,g,h) <=> (f or g) and (-f or h): expected true, actual false")
	}
}

// TestOperations implements the same tests than the bddtest program in the
// Buddy distribution. It uses function Allsat for checking that all assignments
// are detected.
func TestOperations(t *testing.T) {
	varnum := 4
	bdd, _ := New(varnum, Nodesize(1000), Cachesize(1000))
	check := func(x Node) error {
		allsatBDD := x
		allsatSumBDD := bdd.False()
		// Calculate whole set of asignments and remove all assignments
		// from original set
		bdd.Allsat(func(varset []int) error {
			x := bdd.True()
			for k, v := range varset {
				switch v {
				case 0:
					x = bdd.And(x, bdd.NIthvar(k))
				case 1:
					x = bdd.And(x, bdd.Ithvar(k))
				}
			}
			t.Logf("Checking bdd with %-4s assignments\n", bdd.Satcount(x))
			// Sum up all assignments
			allsatSumBDD = bdd.Or(allsatSumBDD, x)
			// Remove assignment from initial set
			allsatBDD = bdd.Apply(allsatBDD, x, OPdiff)
			return nil
		}, x)

		// Now the summed set should be equal to the original set and the
		// subtracted set should be empty
		if !bdd.Equal(allsatSumBDD, x) {
			return fmt.Errorf("AllSat sum is not the initial BDD")
		}

		if !bdd.Equal(allsatBDD, bdd.False()) {
			return fmt.Errorf("AllSat is not False")
		}
		return nil
	}

	a := bdd.Ithvar(0)
	b := bdd.Ithvar(1)
	c := bdd.Ithvar(2)
	d := bdd.Ithvar(3)
	na := bdd.NIthvar(0)
	nb := bdd.NIthvar(1)
	nc := bdd.NIthvar(2)
	nd := bdd.NIthvar(3)

	check(bdd.True())

	check(bdd.False())

	// a & b | !a & !b
	check(bdd.Or(bdd.And(a, b), bdd.And(na, nb)))

	// a & b | c & d
	check(bdd.Or(bdd.And(a, b), bdd.And(c, d)))

	// a & !b | a & !d | a & b & !c
	check(bdd.Or(bdd.And(a, nb), bdd.And(a, nd), bdd.And(a, b, nc)))

	for i := 0; i < varnum; i++ {
		check(bdd.Ithvar(i))
		check(bdd.NIthvar(i))
	}

	set := bdd.True()
	for i := 0; i < 50; i++ {
		v := rand.Intn(varnum)
		s := rand.Intn(2)
		o := rand.Intn(2)

		if o == 0 {
			if s == 0 {
				set = bdd.And(set, bdd.Ithvar(v))
			} else {
				set = bdd.And(set, bdd.NIthvar(v))
			}
		} else {
			if s == 0 {
				set = bdd.And(set, bdd.Ithvar(v))
			} else {
				set = bdd.And(set, bdd.NIthvar(v))
			}
		}

		check(set)
	}
}
