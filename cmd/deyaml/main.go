package main

import (
	"fmt"
	"io"
	"os"

	"github.com/codahale/deyaml/pkg/deyaml"
)

func input() (io.ReadCloser, error) {
	if len(os.Args) > 1 {
		return os.Open(os.Args[1])
	}
	return os.Stdin, nil
}

func main() {
	f, err := input()
	if err != nil {
		panic(err)
	}
	defer func() { _ = f.Close() }()

	// parse into k8s objects
	objects, err := deyaml.DeserializeYAML(f)
	if err != nil {
		panic(err)
	}

	// find and print all the type package paths
	packages, aliases := deyaml.CollectImports(objects)
	if len(packages) > 0 {
		fmt.Println("import (")
		for _, v := range packages {
			if alias := aliases[v]; alias != "" {
				fmt.Printf("\t%s %#v\n", alias, v)
			} else {
				fmt.Printf("\t%#v\n", v)
			}
		}
		fmt.Println(")")
		fmt.Println()
	}

	// pretty print the results
	fmt.Printf("var objects = %# v\n\n", deyaml.Formatter(objects, aliases))
}
