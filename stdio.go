// Copyright 2021. Silvano DAL ZILIO.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package rudd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"text/tabwriter"
)

// stats returns information about the BDD
func (b *BDD) stats() string {
	res := fmt.Sprintf("Varnum:     %d\n", b.varnum)
	res += fmt.Sprintf("Allocated:  %d\n", len(b.nodes))
	res += fmt.Sprintf("Produced:   %d\n", b.produced)
	r := (float64(b.freenum) / float64(len(b.nodes))) * 100
	res += fmt.Sprintf("Free:       %d  (%.3g %%)\n", b.freenum, r)
	res += fmt.Sprintf("Used:       %d  (%.3g %%)", len(b.nodes)-b.freenum, (100.0 - r))
	return res
}

func (b *BDD) gcstats() string {
	res := fmt.Sprintf("# of GC:    %d\n", len(b.gchistory))
	allocated := int(b.setfinalizers)
	reclaimed := int(b.calledfinalizers)
	for _, g := range b.gchistory {
		allocated += g.setfinalizers
		reclaimed += g.calledfinalizers
	}
	res += fmt.Sprintf("Ext. refs:  %d\n", allocated)
	res += fmt.Sprintf("Reclaimed:  %d", reclaimed)
	return res
}

// PrintStats outputs a textual representation of the BDD statistics.
func (b *BDD) PrintStats() {
	fmt.Println("==============")
	fmt.Println(b.stats())
	fmt.Println("==============")
	fmt.Println(b.gcstats())
	if _DEBUG {
		fmt.Println("==============")
		fmt.Println(b.cacheStat)
		fmt.Println("==============")
		b.logTable()
	}
	fmt.Println("==============")
}

// ******************************************************************************************************

// Print returns a one-line description of node n.
func (b *BDD) Print(n Node) string {
	if b.error != nil {
		return fmt.Sprintf("node %d: error %s\n", *n, b.error)
	}
	// We build a map between the nodes in the BDD and the vertices in the
	// result. We take advantage of this to compute the number of transitions
	// and vertices in the graph.
	if *n == 0 {
		return "False"
	}
	if *n == 1 {
		return "True"
	}
	if *n < 0 {
		return "Error"
	}
	if *n >= len(b.nodes) {
		return fmt.Sprintf("Error (%d not a valid index)", *n)
	}
	if b.nodes[*n].low == -1 {
		return fmt.Sprintf("Error (node %d[%d] undefined)", *n, b.nodes[*n].level)
	}
	return fmt.Sprintf("(%d[%d] ? %d : %d)", *n, b.nodes[*n].level, b.nodes[*n].low, b.nodes[*n].high)
}

// PrintSet outputs a textual representation of the BDD with root n.
func (b *BDD) PrintSet(n Node) {
	b.print(os.Stdout, n)
}

// PrintAll prints the totally of the BDD table on the standard output
func (b *BDD) PrintAll() {
	b.printAll(os.Stdout)
}

func (b *BDD) print(w io.Writer, n Node) error {
	if b.error != nil {
		fmt.Fprintf(w, "ERROR: %s\n", b.error)
		return b.error
	}
	// We build a map between the nodes in the BDD and the vertices in the
	// result. We take advantage of this to compute the number of transitions
	// and vertices in the graph.
	cnodes := b.markcount(*n)
	nodes := make([]int, cnodes+2)
	nodes[0] = 0
	nodes[1] = 1
	counter := 2
	if *n == 0 {
		fmt.Fprintln(w, "False")
		return nil
	}
	if *n == 1 {
		fmt.Fprintln(w, "True")
		return nil
	}
	fmt.Fprintf(w, "node: %d\n", *n)
	for i := 2; i < len(b.nodes); i++ {
		if b.ismarked(i) {
			b.unmarknode(i)
			nodes[counter] = i
			counter++
		}
	}
	b.print_string(w, nodes)
	return nil
}

func (b *BDD) printAll(w io.Writer) error {
	// if b.error != nil {
	// 	fmt.Fprintf(w, "ERROR: %s\n", b.error)
	// 	return b.error
	// }
	// We build a map between the nodes in the BDD and the vertices in the
	// result. We take advantage of this to compute the number of transitions
	// and vertices in the graph.
	nodes := make([]int, 2)
	nodes[0] = 0
	nodes[1] = 1
	for i := 2; i < len(b.nodes); i++ {
		if b.nodes[i].low != -1 {
			nodes = append(nodes, i)
		}
	}
	b.print_string(w, nodes)
	return nil
}

func (b *BDD) print_string(w io.Writer, nodes []int) {
	tw := tabwriter.NewWriter(w, 0, 0, 0, ' ', 0)
	sort.Ints(nodes)
	for _, n := range nodes {
		if n > 1 {
			fmt.Fprintf(tw, "%d\t[%d\t] ? \t%d\t : %d\n", n, b.nodes[n].level, b.nodes[n].low, b.nodes[n].high)
		}
	}
	tw.Flush()
}

// ******************************************************************************************************

// Example of AUT output for `nd` with properties on states
//
// des(0,8,4)
// (0,"S.`p0`",0)
// (0,"E.`t0`",1)
// (1,"S.`p1` S.`p2` S.`p3`",1)
// (1,"E.`t1`",2)
// (2,"S.`p3` S.`p4`",2)
// (2,"E.`t2`",3)
// (3,"S.`p5`",3)
// (3,"E.`t3`",0)

// PrintAut prints a textual, graph-like, description representing the BDD with
// root n using the AUT format. The file can be displayed using the nd tool.
func (b *BDD) PrintAut(n Node) {
	b.printAut(bufio.NewWriter(os.Stdout), n)
}

// PrintAllAut prints a textual, graph-like, description of all the nodes in the
// BDD using the AUT format. The file can be displayed using the nd tool.
func (b *BDD) PrintAllAut() {
	b.printAllAut(bufio.NewWriter(os.Stdout))
}

func (b *BDD) FPrintAut(filename string, n Node) error {
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
	return b.printAut(bufio.NewWriter(out), n)
}

func (b *BDD) FPrintAllAut(filename string) error {
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
	return b.printAllAut(bufio.NewWriter(out))
}

func (b *BDD) printAut(w *bufio.Writer, n Node) error {
	if b.error != nil {
		fmt.Fprintf(w, "ERROR: %s\n", b.error)
		return b.error
	}
	// We build a map between the nodes in the BDD and the vertices in the
	// result. We take advantage of this to compute the number of transitions
	// and vertices in the graph.
	cnodes := b.markcount(*n)
	nodes := make(map[int]int, cnodes)
	nodes[0] = 0
	nodes[1] = 1
	counter := 2
	for i := 2; i < len(b.nodes); i++ {
		if b.ismarked(i) {
			b.unmarknode(i)
			nodes[i] = counter
			counter++
		}
	}
	b.print_aut(w, nodes)
	return nil
}

func (b *BDD) printAllAut(w *bufio.Writer) error {
	if b.error != nil {
		fmt.Fprintf(w, "ERROR: %s\n", b.error)
		return b.error
	}
	// We build a map between the nodes in the BDD and the vertices in the
	// result. We take advantage of this to compute the number of transitions
	// and vertices in the graph.
	nodes := make(map[int]int)
	nodes[0] = 0
	nodes[1] = 1
	counter := 2
	for i := 2; i < len(b.nodes); i++ {
		if b.nodes[i].low != -1 {
			nodes[i] = counter
			counter++
		}
	}
	b.print_aut(w, nodes)
	return nil
}

func (b *BDD) print_aut(w *bufio.Writer, nodes map[int]int) {
	cnodes := len(nodes)
	fmt.Fprintf(w, "des(0,%d,%d)\n", 3*cnodes-4, cnodes)
	fmt.Fprintln(w, "(0, \"S."+"`"+"False"+"`"+"\", 0)")
	fmt.Fprintln(w, "(1, \"S."+"`"+"True"+"`"+"\", 1)")
	for k, v := range nodes {
		if k > 1 {
			fmt.Fprintf(w, "(%d, \"S."+"`"+"%d"+"`"+"\", %[1]d)\n", v, b.nodes[k].level)
			fmt.Fprintf(w, "(%d, \"E."+"`"+"0"+"`"+"\", %d)\n", v, nodes[b.nodes[k].low])
			fmt.Fprintf(w, "(%d, \"E."+"`"+"1"+"`"+"\", %d)\n", v, nodes[b.nodes[k].high])
		}
	}
	w.Flush()
}

// ******************************************************************************************************

// PrintDot prints a graph-like description of the BDD with root n using the DOT
// format.
func (b *BDD) PrintDot(n Node) {
	b.printDot(bufio.NewWriter(os.Stdout), n)
}

func (b *BDD) PrintAllDot() {
	b.printAllDot(bufio.NewWriter(os.Stdout))
}

func (b *BDD) FPrintDot(filename string, n Node) error {
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
	return b.printDot(bufio.NewWriter(out), n)
}

func (b *BDD) FPrintAllDot(filename string) error {
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
	return b.printAllDot(bufio.NewWriter(out))
}

func (b *BDD) printDot(w *bufio.Writer, n Node) error {
	if b.error != nil {
		fmt.Fprintf(w, "ERROR: %s\n", b.error)
		w.Flush()
		return b.error
	}
	// We build a map between the nodes in the BDD and the vertices in the
	// result. We take advantage of this to compute the number of transitions
	// and vertices in the graph.
	cnodes := b.markcount(*n)
	nodes := make([]int, cnodes+2)
	nodes[0] = 0
	nodes[1] = 1
	counter := 2
	for i := 2; i < len(b.nodes); i++ {
		if b.ismarked(i) {
			b.unmarknode(i)
			nodes[counter] = i
			counter++
		}
	}
	b.print_dot(w, nodes)
	return nil
}

func (b *BDD) printAllDot(w *bufio.Writer) error {
	if b.error != nil {
		fmt.Fprintf(w, "ERROR: %s\n", b.error)
		return b.error
	}
	// We build a map between the nodes in the BDD and the vertices in the
	// result. We take advantage of this to compute the number of transitions
	// and vertices in the graph.
	nodes := make([]int, 2)
	nodes[0] = 0
	nodes[1] = 1
	for i := 2; i < len(b.nodes); i++ {
		if b.nodes[i].low != -1 {
			nodes = append(nodes, i)
		}
	}
	b.print_dot(w, nodes)
	return nil
}

// print_dot returns a GraphViz DOT file from a list of nodes. We do not draw
// arcs that go to the constant false.
func (b *BDD) print_dot(w *bufio.Writer, nodes []int) {
	sort.Ints(nodes)
	fmt.Fprintln(w, "digraph G {")
	// fmt.Fprintln(w, "0 [shape=box, label=\"0\", style=filled, shape=box, height=0.3, width=0.3];")
	fmt.Fprintln(w, "1 [shape=box, label=\"1\", style=filled, shape=box, height=0.3, width=0.3];")

	for _, v := range nodes {
		if v > 1 {
			fmt.Fprintf(w, "%d %s\n", v, dotlabel(v, b.level2var[b.nodes[v].level]))
			if b.nodes[v].low != 0 {
				fmt.Fprintf(w, "%d -> %d [style=dotted];\n", v, b.nodes[v].low)
			}
			if b.nodes[v].high != 0 {
				fmt.Fprintf(w, "%d -> %d [style=filled];\n", v, b.nodes[v].high)
			}
		}
	}
	fmt.Fprintln(w, "}")

	w.Flush()
}

func dotlabel(a int, b int32) string {
	return fmt.Sprintf(`[label=<
	<FONT POINT-SIZE="20">%d</FONT>
	<FONT POINT-SIZE="10">[%d]</FONT>
>];`, b, a)
}
