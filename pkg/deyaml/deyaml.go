package deyaml

import (
	"bytes"
	"fmt"
	"go/format"
	"reflect"
	"sort"
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
	packages, aliases := imports(objects)
	_, _ = fmt.Fprintf(buf, "import (\n")
	_, _ = fmt.Fprintf(buf, "\t%#v\n", "k8s.io/apimachinery/pkg/runtime")
	for _, v := range packages {
		if alias := aliases[v]; alias != "" {
			_, _ = fmt.Fprintf(buf, "\t%s %#v\n", alias, v)
		} else {
			_, _ = fmt.Fprintf(buf, "\t%#v\n", v)
		}
	}
	_, _ = fmt.Fprintf(buf, ")\n")

	// pretty-print the objects
	_, _ = fmt.Fprintf(buf, "var objects = %# v\n\n", prettyFormatter(objects, aliases))

	// format the final source code
	src, err := format.Source(buf.Bytes())
	if err != nil {
		return "", err
	}
	return string(src), nil
}

func imports(objects []runtime.Object) ([]string, map[string]string) {
	// collect packages
	m := map[string]bool{}
	for _, object := range objects {
		collectPackages(reflect.ValueOf(object), m)
	}

	packages := make([]string, 0, len(m))
	for k := range m {
		packages = append(packages, k)
	}
	sort.Strings(packages)

	// collect all packages by their last segment
	byName := make(map[string][]string, len(packages))
	for _, v := range packages {
		segments := strings.Split(v, "/")
		n := segments[len(segments)-1]
		byName[n] = append(byName[n], v)
	}

	aliases := make(map[string]string, len(packages))
	for _, packages := range byName {
		// no need to alias unambiguous imports
		if len(packages) == 1 {
			aliases[packages[0]] = ""
			continue
		}

		// use the last two path segments to dedupe
		for _, v := range packages {
			segments := strings.Split(v, "/")
			aliases[v] = strings.Join(segments[len(segments)-2:], "")
		}
	}
	return packages, aliases
}

func collectPackages(v reflect.Value, m map[string]bool) {
	if !v.IsValid() || !nonzero(v) {
		return
	}

	if pkg := v.Type().PkgPath(); pkg != "" {
		m[pkg] = true
	}

	// deref pointers and interfaces
	if v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
		collectPackages(v.Elem(), m)
	}

	// iterate through struct fields
	if v.Kind() == reflect.Struct {
		i := 0
		for i < v.NumField() {
			collectPackages(v.Field(i), m)
			i++
		}
	}
}
