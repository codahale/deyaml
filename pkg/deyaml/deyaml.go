package deyaml

import (
	"io"
	"io/ioutil"
	"reflect"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/yaml"
)

func DeserializeYAML(r io.Reader) (runtime.Object, error) {
	// read YAML from stdin
	y, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// convert YAML to JSON
	j, err := yaml.YAMLToJSON(y)
	if err != nil {
		return nil, err
	}

	// convert decode JSON as unstructured data so we can read the metadata
	got, _, err := unstructured.UnstructuredJSONScheme.Decode(j, nil, nil)
	if err != nil {
		return nil, err
	}
	gvk := got.GetObjectKind().GroupVersionKind()

	// make a scheme with all of k8s's known types in it
	scheme := runtime.NewScheme()
	err = fake.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	// decode the JSON again now that we know the type
	s := json.NewSerializer(json.DefaultMetaFactory, scheme, scheme, false)
	object, _, err := s.Decode(j, &gvk, nil)
	if err != nil {
		return nil, err
	}
	return object, nil
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
