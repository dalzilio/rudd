// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd

import (
	"math/big"
	"testing"
)

// milner is an example of using BDD for state space computation. It is
// directly adapted from the examples in the Buddy distribution. It computes the
// reachable state of a system composed of N cyclers, with an initial BDD size
// of size. For this system, we have an anlytical formula to compute the size of
// the state space.
func milner(tb testing.TB, fast bool, varnum int, options ...func(*configs)) (*BDD, Node) {
	bdd, err := New(varnum*6, options...)
	if err != nil {
		tb.Error(err)
	}
	c := make([]Node, varnum)
	cp := make([]Node, varnum)
	t := make([]Node, varnum)
	tp := make([]Node, varnum)
	h := make([]Node, varnum)
	hp := make([]Node, varnum)

	for n := 0; n < varnum; n++ {
		c[n] = bdd.Ithvar(n * 6)
		cp[n] = bdd.Ithvar(n*6 + 1)
		t[n] = bdd.Ithvar(n*6 + 2)
		tp[n] = bdd.Ithvar(n*6 + 3)
		h[n] = bdd.Ithvar(n*6 + 4)
		hp[n] = bdd.Ithvar(n*6 + 5)
	}

	nvar := make([]int, varnum*3)
	pvar := make([]int, varnum*3)
	for n := 0; n < varnum*3; n++ {
		nvar[n] = n * 2   // normal variables
		pvar[n] = n*2 + 1 // primed variables
	}
	replacer, err := bdd.NewReplacer(pvar, nvar)
	if err != nil {
		tb.Error(err)
	}

	// We create a BDD for the initial state of Milner's cyclers.
	I := bdd.And(c[0], bdd.Not(h[0]), bdd.Not(t[0]))
	for i := 1; i < varnum; i++ {
		I = bdd.And(I, bdd.Not(c[i]), bdd.Not(h[i]), bdd.Not(t[i]))
	}

	// A builds a BDD expressing that all other variables than 'z' is unchanged.
	A := func(x, y []Node, z int) Node {
		res := bdd.True()
		for i := 0; i < varnum; i++ {
			if i != z {
				res = bdd.And(res, bdd.Equiv(x[i], y[i]))
			}
		}
		return res
	}

	// Now we compute the transition relation
	T := bdd.False() // The monolithic transition relation
	for i := 0; i < varnum; i++ {
		P1 := bdd.And(c[i], bdd.Not(cp[i]), tp[i], bdd.Not(t[i]), hp[i], A(c, cp, i), A(t, tp, i), A(h, hp, i))
		P2 := bdd.And(h[i], bdd.Not(hp[i]), cp[(i+1)%varnum], A(c, cp, (i+1)%varnum), A(h, hp, i), A(t, tp, varnum))
		E := bdd.And(t[i], bdd.Not(tp[i]), A(t, tp, i), A(h, hp, varnum), A(c, cp, varnum))
		T = bdd.Or(T, P1, bdd.Or(P2, E))
	}

	// We compute the reachable states.
	R := I // Reachable state space
	normvar := bdd.Makeset(nvar)
	count := 0
	for {
		count++
		prev := R
		if fast {
			R = bdd.Or(bdd.Replace(bdd.AndExist(normvar, R, T), replacer), R)
		} else {
			R = bdd.Or(bdd.Replace(bdd.Exist(bdd.And(R, T), normvar), replacer), R)
		}
		if *prev == *R {
			break
		}
	}
	if _LOGLEVEL > 0 {
		tb.Log("\n", bdd.Stats())
	}
	return bdd, R
}

func TestMilnerSlow(t *testing.T) {
	for _, N := range []int{4, 5, 7, 11} {
		// we choose a small size to stress test garbage collection
		fast, Rfast := milner(t, true, N, Nodesize(100), Cachesize(25), Cacheratio(25))
		slow, Rslow := milner(t, false, N, Nodesize(100), Cachesize(25), Cacheratio(25))
		expected := big.NewInt(int64(N))
		pow := big.NewInt(0)
		pow.SetBit(pow, 4*N+1, 1)
		expected.Mul(expected, pow)
		fastresult := fast.Satcount(Rfast)
		slowresult := slow.Satcount(Rslow)
		if fastresult.Cmp(expected) != 0 || slowresult.Cmp(expected) != 0 {
			t.Errorf("Error in Milner(%d), expected %s, actual %s (fast) and %s (slow)", N, expected, fastresult, slowresult)
		}
	}
}

func Testmilner(t *testing.T) {
	for _, N := range []int{16, 20, 30, 50} {
		// we choose a small size to stress test garbage collection
		bdd, R := milner(t, true, N, Nodesize(100000))
		expected := big.NewInt(int64(N))
		pow := big.NewInt(0)
		pow.SetBit(pow, 4*N+1, 1)
		expected.Mul(expected, pow)
		result := bdd.Satcount(R)
		if result.Cmp(expected) != 0 {
			t.Errorf("Error in Milner(%d), expected %s, actual %s", N, expected, result)
		}
	}
}

func TestMilner80(t *testing.T) {
	N := 80
	tt := func(buddy bool) {
		bdd, R := milner(t, true, N, Nodesize(1000000), Cachesize(250000), Cacheratio(25))
		expected := big.NewInt(int64(N))
		pow := big.NewInt(0)
		pow.SetBit(pow, 4*N+1, 1)
		expected.Mul(expected, pow)
		result := bdd.Satcount(R)
		if result.Cmp(expected) != 0 {
			t.Errorf("Error in Milner(%d), expected %s, actual %s", N, expected, result)
		}
	}
	tt(true)
	tt(false)
}

func BenchmarkMilner150(b *testing.B) {
	// run the milner_system function b.N times
	for n := 0; n < b.N; n++ {
		milner(b, true, 150, Nodesize(1000000), Cachesize(250000), Cacheratio(25))
	}
}

func BenchmarkMilner300(b *testing.B) {
	// run the milner_system function b.N times
	for n := 0; n < b.N; n++ {
		milner(b, true, 300, Nodesize(1000000), Cachesize(250000), Cacheratio(25), Maxnodeincrease(1<<23))
	}
}
