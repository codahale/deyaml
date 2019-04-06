package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"

	"github.com/codahale/deyaml/pkg/deyaml"
)

func main() {
	// parse k8s objects
	objects, err := deyaml.DeserializeYAML(os.Args[1:]...)
	if err != nil {
		panic(err)
	}

	// find and print all the type package paths
	buf := bytes.NewBuffer(nil)
	_, _ = fmt.Fprintf(buf, "package example\n\n")
	packages, aliases := deyaml.CollectImports(objects)
	if len(packages) > 0 {
		_, _ = fmt.Fprintln(buf, "import (")
		for _, v := range packages {
			if alias := aliases[v]; alias != "" {
				_, _ = fmt.Fprintf(buf, "\t%s %#v\n", alias, v)
			} else {
				_, _ = fmt.Fprintf(buf, "\t%#v\n", v)
			}
		}
		_, _ = fmt.Fprintf(buf, ")\n")
	}

	// pretty-print the objects
	_, _ = fmt.Fprintf(buf, "var objects = %# v\n\n", deyaml.Formatter(objects, aliases))

	// format the final source code
	src, err := format.Source(buf.Bytes())
	if err != nil {
		panic(err)
	}
	fmt.Print(string(src))
}
