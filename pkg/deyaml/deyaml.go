package deyaml

import (
	"reflect"
	"sort"
	"strings"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubernetes/pkg/kubectl/scheme"
)

func DeserializeYAML(filenames ...string) ([]interface{}, error) {
	b := resource.NewBuilder(genericclioptions.NewConfigFlags(true))
	var objs []interface{}
	res := b.WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		ContinueOnError().
		DefaultNamespace().
		FilenameParam(true, &resource.FilenameOptions{
			Filenames: filenames,
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

func CollectImports(objects []interface{}) ([]string, map[string]string) {
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
