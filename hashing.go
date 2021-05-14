// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd

// Hash functions

func _TRIPLE(a, b, c, len int) int {
	return int(_PAIR64(uint64(c), _PAIR(a, b, len), uint64(len)))
}

// _PAIR is a mapping function that maps (bijectively) a pair of integer (a, b)
// into a unique integer. It is therefore a perfect hash: no collisions
func _PAIR(a, b, len int) uint64 {
	return (((uint64(a+b) * uint64(a+b+1)) / 2) + uint64(a)) % uint64(len)
}

func _PAIR64(a, b, len uint64) uint64 {
	return (((((a + b) % len) * ((a + b + 1) % len)) / 2) + a) % len
}

// ************************************************************

// The hash function for nodes is #(level, low, high)

func (b *buddy) ptrhash(n int) int {
	return _TRIPLE(int(b.nodes[n].level), b.nodes[n].low, b.nodes[n].high, len(b.nodes))
}

func (b *buddy) nodehash(level int32, low, high int) int {
	return _TRIPLE(int(level), low, high, len(b.nodes))
}

// ************************************************************

// The hash function for operation Not(n) is simply n.

func (b *buddy) matchnot(n int) int {
	entry := b.applycache.table[n%len(b.applycache.table)]
	if entry.a == n && entry.c == int(op_not) {
		return entry.res
	}
	return -1
}

func (b *buddy) setnot(n int, res int) int {
	if res < 0 {
		b.seterror("problem in call to not")
		return -1
	}
	b.applycache.table[n%len(b.applycache.table)] = cacheData{
		a:   n,
		c:   int(op_not),
		res: res,
	}
	return res
}

// ************************************************************

// The hash function for Apply is #(left, right, applycache.op).

func (b *buddy) matchapply(left, right int) int {
	entry := b.applycache.table[_TRIPLE(left, right, int(b.applycache.op), len(b.applycache.table))]
	if entry.a == left && entry.b == right && entry.c == int(b.applycache.op) {
		return entry.res
	}
	return -1
}

func (b *buddy) setapply(left, right, res int) int {
	if res < 0 {
		b.seterror("problem in call to apply(%d,%d,%s)", left, right, b.applycache.op)
		return -1
	}
	b.applycache.table[_TRIPLE(left, right, int(b.applycache.op), len(b.applycache.table))] = cacheData{
		a:   left,
		b:   right,
		c:   int(b.applycache.op),
		res: res,
	}
	return res
}

// ************************************************************

// The hash function for ITE is #(f,g,h).

func (b *buddy) matchite(f, g, h int) int {
	entry := b.itecache.table[_TRIPLE(f, g, h, len(b.itecache.table))]
	if entry.a == f && entry.b == g && entry.c == h {
		return entry.res
	}
	return -1
}

func (b *buddy) setite(f, g, h, res int) int {
	if res < 0 {
		b.seterror("problem in call to ite")
		return -1
	}
	b.itecache.table[_TRIPLE(f, g, h, len(b.itecache.table))] = cacheData{
		a:   f,
		b:   g,
		c:   h,
		res: res,
	}
	return res
}

// ************************************************************

// The hash function for quantification is simply n.

func (b *buddy) matchquant(n int) int {
	entry := b.quantcache.table[n%len(b.quantcache.table)]
	if entry.a == n && entry.c == b.quantcache.id {
		return entry.res
	}
	return -1
}

func (b *buddy) setquant(n int, res int) int {
	if res < 0 {
		b.seterror("problem in call to quantification")
		return -1
	}
	b.quantcache.table[n%len(b.quantcache.table)] = cacheData{
		a:   n,
		c:   b.quantcache.id,
		res: res,
	}
	return res
}

// ************************************************************

// The hash function for AppEx is #(left, right)

func (b *buddy) matchappex(left, right int) int {
	entry := b.appexcache.table[int(_PAIR(left, right, len(b.appexcache.table)))]
	if entry.a == left && entry.b == right && entry.c == b.appexcache.id {
		return entry.res
	}
	return -1
}

func (b *buddy) setappex(left, right, res int) int {
	if res < 0 {
		b.seterror("problem in call to appex")
		return -1
	}
	b.appexcache.table[int(_PAIR(left, right, len(b.appexcache.table)))] = cacheData{
		a:   left,
		b:   right,
		c:   b.appexcache.id,
		res: res,
	}
	return res
}

// ************************************************************

// The hash function for operation Replace(n) is simply n.

func (b *buddy) matchreplace(n int) int {
	entry := b.replacecache.table[n%len(b.replacecache.table)]
	if entry.a == n && entry.c == b.replacecache.id {
		return entry.res
	}
	return -1
}

func (b *buddy) setreplace(n int, res int) int {
	if res < 0 {
		b.seterror("problem in call to replace")
		return -1
	}
	b.replacecache.table[n%len(b.replacecache.table)] = cacheData{
		a:   n,
		c:   b.replacecache.id,
		res: res,
	}
	return res
}
