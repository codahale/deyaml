package main

import (
	"fmt"
	"os"

	"github.com/codahale/deyaml/pkg/deyaml"
)

func main() {
	// parse k8s objects
	objects, err := deyaml.Parse(os.Args[1:]...)
	if err != nil {
		panic(err)
	}

	// generate source code
	src, err := deyaml.GenerateExample(objects)
	if err != nil {
		panic(err)
	}
	fmt.Println(src)
}
