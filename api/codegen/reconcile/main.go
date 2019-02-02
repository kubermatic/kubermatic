// The following directive is necessary to make the package coherent:

package main

import (
	"bytes"
	"go/format"
	"io/ioutil"
	"log"
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
			},
			{
				ResourceName:       "StatefulSet",
				ImportAlias:        "appsv1",
				ResourceImportPath: "k8s.io/api/apps/v1",
			},
			{
				ResourceName:       "VerticalPodAutoscaler",
				ImportAlias:        "autoscalingv1beta1",
				ResourceImportPath: "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta1",
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

var (
	tplFuncs = map[string]interface{}{
		"reconcileFunc": reconcileFunc,
	}
	reconcileAllTemplate = template.Must(template.New("").Funcs(tplFuncs).Funcs(sprig.TxtFuncMap()).Parse(`// This file is generated. DO NOT EDIT.
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
{{ reconcileFunc .ResourceName .ImportAlias }} 
{{- end }}

`))
)

type reconcileFunctionData struct {
	ResourceName       string
	ResourceImportPath string
	ImportAlias        string
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

var reconcileFunctionTemplate = template.Must(template.New("").Parse(`// {{ .ResourceName }}ObjectWrapper adds a wrapper so the {{ .ResourceName }}Creator matches ObjectCreator
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
func Reconcile{{ .ResourceName }}s(creators []{{ .ResourceName }}Creator, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, wrapper ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &{{ .ImportAlias }}.{{ .ResourceName }}{})
	if err != nil {
		return fmt.Errorf("failed to get {{ .ResourceName }} informer: %v", err)
	}

	for _, create := range creators {
		createObject := {{ .ResourceName }}ObjectWrapper(create)
		for _, wrap := range wrapper {
			createObject = wrap(createObject)
		}

		if err := EnsureObject(namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure {{ .ResourceName }}: %v", err)
		}
	}

	return nil
}

`))
