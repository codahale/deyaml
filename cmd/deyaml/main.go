package main

import (
	"fmt"
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
