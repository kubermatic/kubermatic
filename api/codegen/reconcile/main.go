// The following directive is necessary to make the package coherent:

package main

import (
	"bytes"
	"go/format"
	"io/ioutil"
	"log"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
)

func main() {
	data := struct {
		Resources []reconcileFunctionData
	}{
		Resources: []reconcileFunctionData{
			{
				ResourceName:       "Service",
				ImportAlias:        "corev1",
				ResourceImportPath: "k8s.io/api/core/v1",
				UseNamedObject:     true,
			},
			{
				ResourceName:   "Secret",
				ImportAlias:    "corev1",
				UseNamedObject: true,
				// Don't specify ResourceImportPath so this block does not create a new import line in the generated code
			},
			{
				ResourceName:   "ConfigMap",
				ImportAlias:    "corev1",
				UseNamedObject: true,
				// Don't specify ResourceImportPath so this block does not create a new import line in the generated code
			},
			{
				ResourceName: "ServiceAccount",
				ImportAlias:  "corev1",
				// Don't specify ResourceImportPath so this block does not create a new import line in the generated code
			},
			{
				ResourceName:       "StatefulSet",
				ImportAlias:        "appsv1",
				ResourceImportPath: "k8s.io/api/apps/v1",
				UseNamedObject:     true,
			},
			{
				ResourceName:   "Deployment",
				ImportAlias:    "appsv1",
				UseNamedObject: true,
				// Don't specify ResourceImportPath so this block does not create a new import line in the generated code
			},
			{
				ResourceName:       "PodDisruptionBudget",
				ImportAlias:        "policyv1beta1",
				ResourceImportPath: "k8s.io/api/policy/v1beta1",
			},
			{
				ResourceName:       "VerticalPodAutoscaler",
				ImportAlias:        "autoscalingv1beta2",
				ResourceImportPath: "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2",
			},
			{
				ResourceName:       "ClusterRoleBinding",
				ImportAlias:        "rbacv1",
				ResourceImportPath: "k8s.io/api/rbac/v1",
			},
			{
				ResourceName: "Role",
				ImportAlias:  "rbacv1",
				// Don't specify ResourceImportPath so this block does not create a new import line in the generated code
				UseNamedObject: true,
			},
			{
				ResourceName: "RoleBinding",
				ImportAlias:  "rbacv1",
				// Don't specify ResourceImportPath so this block does not create a new import line in the generated code
			},
			{
				ResourceName:       "CustomResourceDefinition",
				ImportAlias:        "apiextensionsv1beta1",
				ResourceImportPath: "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1",
				UseNamedObject:     true,
			},
		},
	}

	buf := &bytes.Buffer{}
	if err := reconcileAllTemplate.Execute(buf, data); err != nil {
		log.Fatal(err)
	}

	fmtB, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	if err := ioutil.WriteFile("zz_generated_reconcile.go", fmtB, 0644); err != nil {
		log.Fatal(err)
	}
}

func lowercaseFirst(str string) string {
	return strings.ToLower(string(str[0])) + str[1:]
}

var (
	reconcileAllTplFuncs = map[string]interface{}{
		"reconcileFunc":      reconcileFunc,
		"namedReconcileFunc": namedReconcileFunc,
	}
	reconcileAllTemplate = template.Must(template.New("").Funcs(reconcileAllTplFuncs).Funcs(sprig.TxtFuncMap()).Parse(`// This file is generated. DO NOT EDIT.
package resources

import (
	"fmt"

	informerutil "github.com/kubermatic/kubermatic/api/pkg/util/informer"

  "k8s.io/apimachinery/pkg/runtime"
	ctrlruntimecache "sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
{{ range .Resources }}
{{- if .ResourceImportPath }}
	{{ .ImportAlias }} "{{ .ResourceImportPath }}"
{{- end }}
{{- end }}
)

{{ range .Resources }}
{{ if .UseNamedObject }}
{{ namedReconcileFunc .ResourceName .ImportAlias }}
{{ else }}
{{ reconcileFunc .ResourceName .ImportAlias }}
{{ end }}
{{- end }}

`))
)

type reconcileFunctionData struct {
	ResourceName       string
	ResourceImportPath string
	ImportAlias        string
	UseNamedObject     bool
}

func reconcileFunc(resourceName, importAlias string) (string, error) {
	b := &bytes.Buffer{}
	err := reconcileFunctionTemplate.Execute(b, struct {
		ResourceName string
		ImportAlias  string
	}{
		ResourceName: resourceName,
		ImportAlias:  importAlias,
	})
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

func namedReconcileFunc(resourceName, importAlias string) (string, error) {
	b := &bytes.Buffer{}
	err := namedReconcileFunctionTemplate.Execute(b, struct {
		ResourceName string
		ImportAlias  string
	}{
		ResourceName: resourceName,
		ImportAlias:  importAlias,
	})
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

var (
	reconcileFunctionTplFuncs = map[string]interface{}{
		"lowercaseFirst": lowercaseFirst,
	}

	reconcileFunctionTemplate = template.Must(template.New("").Funcs(reconcileFunctionTplFuncs).Parse(`// {{ .ResourceName }}Creator defines an interface to create/update {{ .ResourceName }}s
type {{ .ResourceName }}Creator = func(existing *{{ .ImportAlias }}.{{ .ResourceName }}) (*{{ .ImportAlias }}.{{ .ResourceName }}, error)

// {{ .ResourceName }}ObjectWrapper adds a wrapper so the {{ .ResourceName }}Creator matches ObjectCreator
// This is needed as golang does not support function interface matching
func {{ .ResourceName }}ObjectWrapper(create {{ .ResourceName }}Creator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*{{ .ImportAlias }}.{{ .ResourceName }}))
		}
		return create(&{{ .ImportAlias }}.{{ .ResourceName }}{})
	}
}

// Reconcile{{ .ResourceName }}s will create and update the {{ .ResourceName }}s coming from the passed {{ .ResourceName }}Creator slice
func Reconcile{{ .ResourceName }}s(creators []{{ .ResourceName }}Creator, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, objectModifiers ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &{{ .ImportAlias }}.{{ .ResourceName }}{})
	if err != nil {
		return fmt.Errorf("failed to get {{ .ResourceName }} informer: %v", err)
	}

	for _, create := range creators {
		createObject := {{ .ResourceName }}ObjectWrapper(create)
		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureObject(namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure {{ .ResourceName }}: %v", err)
		}
	}

	return nil
}

`))
)

var namedReconcileFunctionTemplate = template.Must(template.New("").Funcs(reconcileFunctionTplFuncs).Parse(`// {{ .ResourceName }}Creator defines an interface to create/update {{ .ResourceName }}s
type {{ .ResourceName }}Creator = func(existing *{{ .ImportAlias }}.{{ .ResourceName }}) (*{{ .ImportAlias }}.{{ .ResourceName }}, error)

// Named{{ .ResourceName }}CreatorGetter returns the name of the resource and the corresponding creator function
type Named{{ .ResourceName }}CreatorGetter = func() (name string, create {{ .ResourceName }}Creator)

// {{ .ResourceName }}ObjectWrapper adds a wrapper so the {{ .ResourceName }}Creator matches ObjectCreator
// This is needed as golang does not support function interface matching
func {{ .ResourceName }}ObjectWrapper(create {{ .ResourceName }}Creator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*{{ .ImportAlias }}.{{ .ResourceName }}))
		}
		return create(&{{ .ImportAlias }}.{{ .ResourceName }}{})
	}
}

// Reconcile{{ .ResourceName }}s will create and update the {{ .ResourceName }}s coming from the passed {{ .ResourceName }}Creator slice
func Reconcile{{ .ResourceName }}s(namedGetters []Named{{ .ResourceName }}CreatorGetter, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, objectModifiers ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &{{ .ImportAlias }}.{{ .ResourceName }}{})
	if err != nil {
		return fmt.Errorf("failed to get {{ .ResourceName }} informer: %v", err)
	}

	for _, get := range namedGetters {
		name, create := get()
		createObject := {{ .ResourceName }}ObjectWrapper(create)
		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(name, namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure {{ .ResourceName }}: %v", err)
		}
	}

	return nil
}

`))
