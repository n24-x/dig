package dig

import (
	"fmt"
	"io"
	"strconv"

	"github.com/n24-x/dig/internal/dot"
)

var _ func(io.Writer, *dot.Graph) = visualizeGraph
var _ func(io.Writer, *dot.Group) = visualizeGroup
var _ func(io.Writer, int, *dot.Constructor) = visualizeConstructor

func visualizeGraph(w io.Writer, dg *dot.Graph) {
	w.Write([]byte("digraph {\n\trankdir=RL;\n\tgraph [compound=true];\n"))
	for _, g := range dg.Groups {
		visualizeGroup(w, g)
	}
	for idx, ctor := range dg.Constructors {
		visualizeConstructor(w, idx, ctor)
	}
	for _, f := range dg.Failed.TransitiveFailures {
		fmt.Fprintf(
			w,
			"\t%s [color=orange];\n",
			strconv.Quote(f.String()),
		)
	}
	for _, f := range dg.Failed.RootCauses {
		fmt.Fprintf(
			w,
			"\t%s [color=red];\n",
			strconv.Quote(f.String()),
		)
	}
	w.Write([]byte("}"))
}

// "[type=*main.t3 group=foo]" [shape=diamond label=<*main.t3<BR /><FONT POINT-SIZE="10">Group: foo</FONT>>];
// "[type=*main.t3 group=foo]" -> "*main.t3[group=foo]0";
// "[type=*main.t3 group=foo]" -> "*main.t3[group=foo]1";
// "[type=*main.t3 group=foo]" -> "*main.t3[group=foo]2";

// visualizeGroup writes the DOT representation of the given g to w.
func visualizeGroup(w io.Writer, g *dot.Group) {
	fmt.Fprintf(
		w,
		"\t%s [%s];\n",
		strconv.Quote(g.String()),
		g.Attributes(),
	)
	for _, r := range g.Results {
		fmt.Fprintf(
			w,
			"\t%s -> %s;\n",
			strconv.Quote(g.String()),
			strconv.Quote(r.String()),
		)
	}
}

//	subgraph cluster_1 {
//	    label = "main";
//	    constructor_1 [shape=plaintext label="main.func2"];
//	    color="red"
//	    "*main.t2" [label=<*main.t2>];
//	}
//
// constructor_1 -> "*main.t1" [ltail=cluster_1];

// visualizeConstructor writes the DOT representation of the given
// constructor ctor to w.
func visualizeConstructor(w io.Writer, index int, ctor *dot.Constructor) {
	fmt.Fprintf(w, "\tsubgraph cluster_%d {\n", index)
	if ctor.Package != "" {
		fmt.Fprintf(w, "\t\tlabel = %s;\n", strconv.Quote(ctor.Package))
	}
	fmt.Fprintf(
		w,
		"\t\tconstructor_%d [shape=plaintext label=%s];\n",
		index,
		strconv.Quote(ctor.Name),
	)

	if ctor.ErrorType != 0 {
		fmt.Fprintf(w, "\t\tcolor=%s;\n", ctor.ErrorType.Color())
	}

	for _, r := range ctor.Results {
		fmt.Fprintf(
			w,
			"\t\t%s [shape=ellipse, %s];\n",
			strconv.Quote(r.String()),
			r.Attributes(),
		)
	}

	fmt.Fprintf(w, "\t}\n")

	for _, p := range ctor.Params {
		var optionalStyle string
		if p.Optional {
			optionalStyle = " style=dashed"
		}

		fmt.Fprintf(
			w,
			"\tconstructor_%d -> %s [ltail=cluster_%d%s];\n",
			index,
			strconv.Quote(p.String()),
			index,
			optionalStyle,
		)
	}

	for _, p := range ctor.GroupParams {
		fmt.Fprintf(
			w,
			"\tconstructor_%d -> %s [ltail=cluster_%d];\n",
			index,
			strconv.Quote(p.String()),
			index,
		)
	}
}
