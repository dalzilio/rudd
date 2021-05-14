// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd

import (
	"fmt"
	"math/rand"
	"testing"
)

//********************************************************************************************

func TestMinus(t *testing.T) {
	var minusTests = []struct {
		p, q, r  int32
		expected int32
	}{
		{3, 2, 3, 2},
		{4, 4, 4, 4},
		{2, 3, 3, 2},
		{3, 2, 2, 2},
		{3, 3, 2, 2},
		{1, 2, 3, 1},
	}
	for _, tt := range minusTests {
		actual := min3(tt.p, tt.q, tt.r)
		if actual != tt.expected {
			t.Errorf("minus3(%d, %d, %d): expected %d, actual %d", tt.p, tt.q, tt.r, tt.expected, actual)
		}
	}
}

//********************************************************************************************

func TestIte_1(t *testing.T) {
	bdd := Buddy(5000, 50)
	bdd.SetVarnum(4)
	n1 := bdd.Makeset([]int{0, 2, 3})
	n2 := bdd.Makeset([]int{0, 3})
	actual := bdd.Equiv(bdd.Ite(n1, n2, bdd.Not(n2)), bdd.Or(bdd.And(n1, n2), bdd.And(bdd.Not(n1), bdd.Not(n2))))
	bdd.PrintDot("aa")
	if actual != bdd.True() {
		t.Errorf("ite(f,g,h) <=> (f or g) and (-f or h): expected true, actual false")
	}
}

//********************************************************************************************

// TestOperations implements the same tests than the bddtest program in the
// Buddy distribution. It uses function Allsat for checking that all assignments
// are detected.

func TestOperations(t *testing.T) {
	bdd := Buddy(1000, 1000)
	bdd.SetVarnum(4)
	varnum := 4

	test1_check := func(x Node) error {
		allsatBDD := x
		allsatSumBDD := bdd.False()
		// Calculate whole set of asignments and remove all assignments
		// from original set
		bdd.Allsat(x, func(varset []int) error {
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
		})

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

	test1_check(bdd.True())

	test1_check(bdd.False())

	// a & b | !a & !b
	test1_check(bdd.Or(bdd.And(a, b), bdd.And(na, nb)))

	// a & b | c & d
	test1_check(bdd.Or(bdd.And(a, b), bdd.And(c, d)))

	// a & !b | a & !d | a & b & !c
	test1_check(bdd.Or(bdd.And(a, nb), bdd.And(a, nd), bdd.And(a, b, nc)))

	for i := 0; i < varnum; i++ {
		test1_check(bdd.Ithvar(i))
		test1_check(bdd.NIthvar(i))
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

		test1_check(set)
	}
}
