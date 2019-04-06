package main

import (
	"fmt"
	"os"

	"github.com/codahale/deyaml/pkg/deyaml"
	"github.com/kr/pretty"
)

func main() {
	// parse into k8s object
	o, err := deyaml.DeserializeYAML(os.Stdin)
	if err != nil {
		panic(err)
	}

	// find and print all the type package paths
	if packages := deyaml.CollectImports(o); len(packages) > 0 {
		fmt.Println("import (")
		for _, v := range packages {
			fmt.Printf("\t%#v\n", v)
		}
		fmt.Println(")")
		fmt.Println()
	}

	// pretty print the results
	fmt.Printf("var data = %# v\n", pretty.Formatter(o))
}
