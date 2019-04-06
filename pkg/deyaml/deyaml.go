package deyaml

import (
	"io"
	"io/ioutil"
	"reflect"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/kubernetes/fake"
)

func DeserializeYAML(r io.Reader) (runtime.Object, error) {
	// make a scheme with all of k8s's known types in it
	scheme := runtime.NewScheme()
	if err := fake.AddToScheme(scheme); err != nil {
		return nil, err
	}

	// make a YAML deserializer
	ser := yaml.NewDecodingSerializer(json.NewSerializer(json.DefaultMetaFactory, scheme, scheme, false))

	// buffer input
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// deserialize object
	obj, _, err := ser.Decode(buf, nil, nil)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func CollectImports(object runtime.Object) []string {
	m := map[string]bool{}
	collectPackages(reflect.ValueOf(object), m)
	packages := make([]string, 0, len(m))
	for k := range m {
		packages = append(packages, k)
	}
	sort.Strings(packages)
	return packages
}

func DedupeImports(packages []string) map[string]string {
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
	return aliases
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
