// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd

// Hash functions

func _TRIPLE(a, b, c, len int) int {
	return int(_PAIR(c, _PAIR(a, b, len), len))
}

// _PAIR is a mapping function that maps (bijectively) a pair of integer (a, b)
// into a unique integer then cast it into a value in the interval [0..len)
// using a modulo operation.
func _PAIR(a, b, len int) int {
	ua := uint64(a)
	ub := uint64(b)
	return int(((((ua + ub) * (ua + ub + 1)) / 2) + (ua)) % uint64(len))
}

// ************************************************************

// The hash function for nodes is #(level, low, high)

func (b *buddy) ptrhash(n int) int {
	return _TRIPLE(int(b.nodes[n].level), b.nodes[n].low, b.nodes[n].high, len(b.nodes))
}

func (b *buddy) nodehash(level int32, low, high int) int {
	return _TRIPLE(int(level), low, high, len(b.nodes))
}
