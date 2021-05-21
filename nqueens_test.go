// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd

import (
	"math/big"
	"testing"
)

// nqueens computes solutions for the N-Queen chess problem and returns
// the number of solutions. It builds a BDD with NxN variables corresponding to
// the squares in the chess board like:
//
//      0 4  8 12
//      1 5  9 13
//      2 6 10 14
//      3 7 11 15
//
// One solution is then that 2,4,11,13 should be true, meaning a queen should be
// placed there:
//
//      . X . .
//      . . . X
//      X . . .
//      . . X .
func nqueens(N int) *big.Int {
	bdd, _ := New(N*N, Nodesize(N*N*256), Cachesize(N*N*64), Cacheratio(30))
	queen := bdd.True()
	X := make([][]Node, N)
	for i := range X {
		X[i] = make([]Node, N)
		for j := range X[i] {
			X[i][j] = bdd.Ithvar(i*N + j)
		}
	}
	// Place a queen in each row
	for i := 0; i < N; i++ {
		e := bdd.False()
		for j := 0; j < N; j++ {
			e = bdd.Or(e, X[i][j])
		}
		queen = bdd.And(queen, e)
	}

	// Build requirements for each variable(field)
	for i := 0; i < N; i++ {
		for j := 0; j < N; j++ {
			// No one in the same column
			a := bdd.True()
			for k := 0; k < N; k++ {
				if k != j {
					a = bdd.And(a, bdd.Imp(X[i][j], bdd.Not(X[i][k])))
				}
			}
			// No one in the same row
			b := bdd.True()
			for k := 0; k < N; k++ {
				if k != i {
					b = bdd.And(b, bdd.Imp(X[i][j], bdd.Not(X[k][j])))
				}
			}
			// No one in the same up-right diagonal
			c := bdd.True()
			for k := 0; k < N; k++ {
				ll := k - i + j
				if ll >= 0 && ll < N {
					if k != i {
						c = bdd.And(c, bdd.Imp(X[i][j], bdd.Not(X[k][ll])))
					}
				}
			}
			// No one in the same down-right diagonal
			d := bdd.True()
			for k := 0; k < N; k++ {
				ll := i + j - k
				if ll >= 0 && ll < N {
					if k != i {
						d = bdd.And(d, bdd.Imp(X[i][j], bdd.Not(X[k][ll])))
					}
				}
			}
			queen = bdd.And(queen, a, b, c, d)
		}
	}
	return bdd.Satcount(queen)
}

func TestNQueens(t *testing.T) {
	var nqueensTests = []struct {
		N        int
		expected int64
	}{
		{4, 2},
		{8, 92},
		{9, 352},
		{10, 724},
	}
	for _, tt := range nqueensTests {
		actual := nqueens(tt.N)
		if actual.Cmp(big.NewInt(tt.expected)) != 0 {
			t.Errorf("Buddy: Error in NQuens(%d), expected %d, actual %s", tt.N, tt.expected, actual)
		}
	}
}

func BenchmarkNQueens(b *testing.B) {
	for n := 0; n < b.N; n++ {
		nqueens(12)
	}
}
