
<!-- PROJECT LOGO -->
<br />
<p align="center">
  <a href="https://github.com/dalzilio/rudd">
    <img src="./docs/rudd1.png" alt="Logo" width="240">
  </a>

  <p align="center">
   RuDD,a library for Binary Decision Diagrams in Go.
    <br />
    <a href="https://github.com/dalzilio/mcc#features"><strong>see what's new »</strong></a>
    <br />
    <!-- <a href="https://github.com/dalzilio/mcc">View Demo</a> -->
  </p>
</p>

## About

RuDD is a Binary Decision Diagram (BDD) library written in pure Go, without the
need for CGo or any other dependendencies. A
[BDD](https://en.wikipedia.org/wiki/Binary_decision_diagram) is a data structure
used to efficiently represent Boolean functions or, equivalently, sets of
Boolean vectors. It has nothing to do with Behaviour Driven Development testing.

RuDD is a direct translation of the
[BuDDy](http://buddy.sourceforge.net/manual/) C-library developed by Jorn
Lind-Nielsen. You can find a high-level description of the algorithms and
data-structures used in this project by looking at ["An Introduction To Binary
Decision Diagrams"](https://www.cs.utexas.edu/~isil/cs389L/bdd.pdf), a Research
Report also distributed as part of the BuDDy distribution. It is a testament to
the many qualities of the BuDDy library, in particular the simplicity (in a good
sense) of its architecture and the legibility of its code.

The source code of RuDD is an almost line-by-line copy of BuDDy (including
reusing part of the same comments for documenting the code), with a few
adaptations to follow some of Go best practices; we even implemented the same
examples than in the BuDDy distribution for benchmarks and regression testing.

Like with [MuDDy](https://github.com/kfl/muddy), a ML interface to BuDDy, we
piggyback on the garbage collection mechanism offered by our host language. We
take care of BDD resizing and memory management directly in the library, but
*external* references to BDD nodes made by user code are automatically managed
by the Go runtime. Unlike MuDDy, we do not provide an interface, but a genuine
reimplementation of BuDDy. As a consequence, we do not suffer from FFI overheads
when calling from Go into C, which is one of the major pain points of working
with Go.  

[![Go Report Card](https://goreportcard.com/badge/github.com/dalzilio/rudd)](https://goreportcard.com/report/github.com/dalzilio/rudd)
[![GoDoc](https://godoc.org/github.com/dalzilio/mcc?status.svg)](https://godoc.org/github.com/dalzilio/rudd)
[![Release](https://img.shields.io/github/v/release/dalzilio/rudd)](https://github.com/dalzilio/rudd/releases)

## Installation 

```
$ go get github.com/dalzilio/rudd
```

## Overview

The main goal of RuDD is to test the performances of a lightweight BDD library
directly implemented in Go, with a focus on implementing symbolic model-checking
tools. At the moment, we provide only a subset of the functionalities defined in
BuDDy, which is enough for our goals. In particular, we do not provide any
method for the dynamic reordering of variables. We also lack support for Finite
Domain Blocks (`fdd`) and Boolean Vectors (`bvec`).

BuDDy is a mature software library, that has been used on several projects, with
performances on par with more complex libraries, such as
[CUDD](https://davidkebo.com/cudd). You can find a comparative study of the
performances for several BDD libraries in this paper
[\[DHJ+2015\]](https://www.tvandijk.nl/pdf/2015setta.pdf). Furthermore,
experiences have shown that there is no significant loss of performance when
using BuDDy from a functional language with garbage collection, compared to
using C or C++
[\[L09\]](https://link.springer.com/content/pdf/10.1007%2F978-3-642-03034-5_3.pdf).
This is one of our motivations in this project. Our first experiments show very
promising results, but we are still lacking a serious study of the performances
of our library.

The library is named after a fresh water fish, the [common
rudd](https://en.wikipedia.org/wiki/Common_rudd) (*Scardinius
erythrophthalmus*), or "gardon rouge" in French, that is stronger and more
resistant that the common roach, with which it is often confused. While it is
sometimes used as bait, its commercial interest is minor. This is certainly a
fitting description for our code ! It is also a valid English word ending with
DD, which is enough to justify our choice.

In the future, we plan to add new features to RuDD and to optimize some of its
internals. For instance with  better  caching strategies. It means that the API
could evolve in future releases but that no functions should disappear or change
significantly.

## References

You may have a look at the documentation for BuDDy (and MuDDy) to get a good
understanding of how the library can be used.

* [\[An97\]](https://www.cs.utexas.edu/~isil/cs389L/bdd.pdf) Henrik Reif
  Andersen. *An Introduction to Binary Decision Diagrams*. Lecture Notes for a
  course on Advanced Algorithms. Technical University of Denmark. 1997.

* [\[L09\]](https://link.springer.com/content/pdf/10.1007%2F978-3-642-03034-5_3.pdf)
  Ken Friis Larsen. [*A MuDDy Experience -– ML Bindings to a BDD
  Library*](https://link.springer.com/chapter/10.1007/978-3-642-03034-5_3)."
  IFIP Working Conference on Domain-Specific Languages. Springer,
  2009.

* [\[DHJ+2015\]](https://www.tvandijk.nl/pdf/2015setta.pdf) Tom van Dijk et al.
  *A comparative study of BDD packages for probabilistic symbolic model
  checking.* International Symposium on Dependable Software Engineering:
  Theories, Tools, and Applications. Springer, 2015.

### Usage

You can find several examples in the `*_test.go` files.

```go
package main

import (
	"rudd"
	"math/big"
)

func main() {
    // create a new BDD with (initially) 10 000 nodes 
    // and a cache size of 5 000
	bdd := rudd.New(10000, 5000)
	bdd.SetVarnum(6)
    // n1 == x2 & x3 & x5
	n1 := bdd.Makeset([]int{2, 3, 5})
    // n2 == x1 | !x3 | x4
	n2 := bdd.Or(bdd.Ithvar(1), bdd.NIthvar(3), bdd.Ithvar(4))
    // n3 == Exists x2,x3,x5 . (n2 & x3)
	n3 := bdd.AppEx(n2, bdd.Ithvar(3), rudd.OPand, n1)
    // you can print the result and also 
    // export a BDD in Graphviz's DOT format
	bdd.Print(n3)
    fmt.Println("Number of sat. assignments: %s\n", bbd.Satcount(n3))
}
```


## Dependencies

The library has no dependencies ouside the standard Go library. The library uses
Go modules and has been tested with Go 1.16.

## License

This software is distributed under the [Apache-2.0
License](https://www.apache.org/licenses/LICENSE-2.0). A copy of the license
agreement is found in the [LICENSE](./LICENSE) file.

The original C code for BuDDy was released under a very permissive license that
is included in the accompanying [NOTICE](./NOTICE) file, together with a list of
the original authors. While the current implementation of RuDD adds some
original work, I expect every redistribution to include the present NOTICE and
acknowledge that some source files and examples have been copied and adapted
from the **BuDDy** Binary Decision Diagrams Library, Package v2.4, Copyright (C)
1996-2002 by Jorn Lind-Nielsen (see <http://buddy.sourceforge.net/>).

## Authors

* **Silvano DAL ZILIO** -  [LAAS/CNRS](https://www.laas.fr/)
