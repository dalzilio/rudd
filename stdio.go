// Copyright (c) 2021 Silvano DAL ZILIO
//
// MIT License

package rudd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"text/tabwriter"
)

var hsizes []string = []string{"k", "M", "G"}

// humanSize returns a human-readable version of a size in bytes
func humanSize(b int, unit uintptr) string {
	b = b * int(unit)
	if b < 0 {
		return "(error)"
	}
	if b < 1000 {
		return fmt.Sprintf("%d B", b)
	}
	c := 0
	s := float64(b) / 1000.0
	for {
		if s < 1000 {
			return fmt.Sprintf("%.1f %sB", s, hsizes[c])
		}
		s = float64(s) / 1000.0
		c++
		if c >= len(hsizes) {
			return fmt.Sprintf("%.1f TB", s)
		}
	}
}

// PrintSet outputs a textual representation of the BDD with root n to the
// standard output. We print all the nodes in b if n is nil.
func (b *BDD) Print(n ...Node) {
	b.print(os.Stdout, n...)
}

func (b *BDD) print(w io.Writer, n ...Node) {
	if mesg := b.Error(); mesg != "" {
		fmt.Fprintf(w, "Error: %s\n", mesg)
		return
	}
	if len(n) == 1 && n[0] != nil {
		if *n[0] == 0 {
			fmt.Fprintln(w, "False")
			return
		}
		if *n[0] == 1 {
			fmt.Fprintln(w, "True")
			return
		}
	}
	// we build a slice of nodes sorted by ids
	nodes := make([][4]int, 0)
	err := b.Allnodes(func(id, level, low, high int) error {
		i := sort.Search(len(nodes), func(i int) bool {
			return nodes[i][0] >= id
		})
		nodes = append(nodes, [4]int{})
		copy(nodes[i+1:], nodes[i:])
		nodes[i] = [4]int{id, level, low, high}
		return nil
	}, n...)
	if err != nil {
		fmt.Fprintln(w, err.Error())
		return
	}
	print_set(w, nodes)
}

func print_set(w io.Writer, nodes [][4]int) {
	tw := tabwriter.NewWriter(w, 0, 0, 0, ' ', 0)
	for _, n := range nodes {
		if n[0] > 1 {
			fmt.Fprintf(tw, "%d\t[%d\t] ? \t%d\t : %d\n", n[0], n[1], n[2], n[3])
		}
	}
	tw.Flush()
}

// PrintDot prints a graph-like description of the BDD with roots in n using the
// DOT format; or the whole Set if n is missing.
func (b *BDD) PrintDot(filename string, n ...Node) error {
	var out *os.File
	var err error
	if filename == "-" {
		out = os.Stdout
	} else {
		out, err = os.Create(filename)
		if err != nil {
			return err
		}
		defer out.Close()
	}
	w := bufio.NewWriter(out)
	if mesg := b.Error(); mesg != "" {
		fmt.Fprintf(w, "Error: %s\n", mesg)
		w.Flush()
		return fmt.Errorf(mesg)
	}
	// we write the result by visiting each node but we never draw edges to the
	// False constant.
	fmt.Fprintln(w, "digraph G {")
	fmt.Fprintln(w, "1 [shape=box, label=\"1\", style=filled, shape=box, height=0.3, width=0.3];")
	_ = b.Allnodes(func(id, level, low, high int) error {
		if id > 1 {
			fmt.Fprintf(w, "%d %s\n", id, dotlabel(id, level))
			if low != 0 {
				fmt.Fprintf(w, "%d -> %d [style=dotted];\n", id, low)
			}
			if high != 0 {
				fmt.Fprintf(w, "%d -> %d [style=filled];\n", id, high)
			}
		}
		return nil
	}, n...)
	fmt.Fprintln(w, "}")
	w.Flush()
	return nil
}

func dotlabel(a int, b int) string {
	return fmt.Sprintf(`[label=<
	<FONT POINT-SIZE="20">%d</FONT>
	<FONT POINT-SIZE="10">[%d]</FONT>
>];`, b, a)
}
