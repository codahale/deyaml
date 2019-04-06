package main

import (
	"fmt"
	"io"
	"os"

	"github.com/codahale/deyaml/pkg/deyaml"
)

func main() {
	var f io.ReadCloser
	if len(os.Args) > 1 {
		o, err := os.Open(os.Args[1])
		if err != nil {
			panic(err)
		}
		f = o
	} else {
		f = os.Stdin
	}
	defer func() { _ = f.Close() }()

	// parse into k8s object
	o, err := deyaml.DeserializeYAML(f)
	if err != nil {
		panic(err)
	}

	// find and print all the type package paths
	packageAliases := map[string]string{}
	if packages := deyaml.CollectImports(o); len(packages) > 0 {
		packageAliases = deyaml.DedupeImports(packages)
		fmt.Println("import (")
		for _, v := range packages {
			if alias := packageAliases[v]; alias != "" {
				fmt.Printf("\t%s %#v\n", alias, v)
			} else {
				fmt.Printf("\t%#v\n", v)
			}
		}
		fmt.Println(")")
		fmt.Println()
	}

	// pretty print the results
	fmt.Printf("var data = %# v\n", deyaml.Formatter(o, packageAliases))
}
