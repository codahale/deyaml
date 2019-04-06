# deyaml

Life's too short to be a YAML farmer. Write some structs.

```
$ deyaml example.yaml

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var objects = []interface {}{
    &appsv1.Deployment{
        TypeMeta:   metav1.TypeMeta{Kind:"Deployment", APIVersion:"apps/v1"},
        ObjectMeta: metav1.ObjectMeta{
            Name:   "wordpress-mysql",
            Labels: map[string]string{"app":"wordpress"},
        },
    },
...
```