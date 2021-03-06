package deyaml

import (
	"bytes"
	"fmt"
	"go/format"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubernetes/pkg/kubectl/scheme"
)

func Parse(filenames ...string) ([]runtime.Object, error) {
	b := resource.NewBuilder(genericclioptions.NewConfigFlags(true))
	var objs []runtime.Object
	res := b.WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		ContinueOnError().
		DefaultNamespace().
		FilenameParam(true, &resource.FilenameOptions{
			Filenames: filenames,
			Recursive: true,
		}).
		Local().
		Do()
	if err := res.Visit(func(info *resource.Info, err error) error {
		objs = append(objs, info.Object)
		return nil
	}); err != nil {
		return nil, err
	}
	return objs, nil
}

func GenerateExample(objects []runtime.Object) (string, error) {
	// find and print all the type package paths
	buf := bytes.NewBuffer(nil)
	_, _ = fmt.Fprintf(buf, "package example\n\n")
	packages := imports(objects)
	_, _ = fmt.Fprintf(buf, "import (\n")
	_, _ = fmt.Fprintf(buf, "\t%#v\n", "k8s.io/apimachinery/pkg/runtime")
	for pkg, alias := range packages {
		if alias != "" {
			_, _ = fmt.Fprintf(buf, "\t%s %#v\n", alias, pkg)
		} else {
			_, _ = fmt.Fprintf(buf, "\t%#v\n", pkg)
		}
	}
	_, _ = fmt.Fprintf(buf, ")\n")

	// pretty-print the objects
	_, _ = fmt.Fprintf(buf, "var objects = %# v\n\n", prettyFormatter(objects, packages))

	// format the final source code
	src, err := format.Source(buf.Bytes())
	if err != nil {
		return "", err
	}
	return string(src), nil
}

func imports(objects []runtime.Object) map[string]string {
	// collect packages
	packages := map[string]bool{}
	for _, object := range objects {
		collectValue(reflect.ValueOf(object), packages)
	}

	// group packages by their last segment
	byName := make(map[string][]string, len(packages))
	for v := range packages {
		segments := strings.Split(v, "/")
		n := segments[len(segments)-1]
		byName[n] = append(byName[n], v)
	}

	// create aliases for dupes
	aliases := make(map[string]string, len(packages))
	for _, dupes := range byName {
		// no need to alias unambiguous imports
		if len(dupes) == 1 {
			aliases[dupes[0]] = ""
			continue
		}

		// use the last path segments to dedupe
		for _, v := range dupes {
			segments := strings.Split(v, "/")
			aliases[v] = strings.Join(segments[len(segments)-2:], "")
		}
	}
	return aliases
}

func collectValue(v reflect.Value, m map[string]bool) {
	// ignore values that won't end up as literals
	if !v.IsValid() || !nonzero(v) {
		return
	}

	// collect the value type
	collectType(v.Type(), m)

	switch v.Kind() {
	// deref pointers and interfaces
	case reflect.Interface, reflect.Ptr:
		collectValue(v.Elem(), m)
	// iterate through struct fields
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			collectValue(v.Field(i), m)
		}
	// collect map k/v types
	case reflect.Map:
		if v.Len() > 0 {
			collectType(v.Type().Key(), m)
			collectType(v.Type().Elem(), m)
		}
	// collect list types
	case reflect.Slice, reflect.Array:
		if v.Len() > 0 {
			collectType(v.Type().Elem(), m)
		}
	}
}

func collectType(t reflect.Type, m map[string]bool) {
	if pkg := t.PkgPath(); pkg != "" {
		m[pkg] = true
	}
}
