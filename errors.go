// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd

import (
	"fmt"
	"log"
)

// Error returns the error status of the BDD.
func (b *bdd) Error() string {
	if b.error == nil {
		return ""
	}
	return b.error.Error()
}

// Errored returns true if there was an error during a computation.
func (b *bdd) Errored() bool {
	return b.error != nil
}

func (b *bdd) seterror(format string, a ...interface{}) Node {
	if b.error != nil {
		format = format + "; " + b.Error()
		b.error = fmt.Errorf(format, a...)
		return nil
	}
	b.error = fmt.Errorf(format, a...)
	if _DEBUG {
		log.Println(b.error)
	}
	return nil
}

// check performs a sanity check prior to accessing a node and return eventual
// error code.
func (b *buddy) checkptr(n Node) error {
	switch {
	case n == nil:
		panic("uncaught error")
	case (*n < 0) || (*n >= len(b.nodes)):
		b.seterror("Illegal acces to node %d", n)
		return b.error
	case (*n >= 2) && (b.nodes[*n].low == -1):
		b.seterror("Illegal acces to node %d", n)
		return b.error
	}
	return nil
}
