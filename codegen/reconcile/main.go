/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// The following directive is necessary to make the package coherent:

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

func main() {
	data := struct {
		Resources []reconcileFunctionData
	}{
		Resources: []reconcileFunctionData{
			{
				ResourceName:       "Namespace",
				ImportAlias:        "corev1",
				ResourceImportPath: "k8s.io/api/core/v1",
			},
			{
				ResourceName:       "Service",
				ImportAlias:        "corev1",
				ResourceImportPath: "k8s.io/api/core/v1",
			},
			{
				ResourceName: "Secret",
				ImportAlias:  "corev1",
				// Don't specify ResourceImportPath so this block does not create a new import line in the generated code
			},
			{
				ResourceName: "ConfigMap",
				ImportAlias:  "corev1",
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
				DefaultingFunc:     "DefaultStatefulSet",
			},
			{
				ResourceName: "Deployment",
				ImportAlias:  "appsv1",
				// Don't specify ResourceImportPath so this block does not create a new import line in the generated code
				DefaultingFunc: "DefaultDeployment",
			},
			{
				ResourceName: "DaemonSet",
				ImportAlias:  "appsv1",
				// Don't specify ResourceImportPath so this block does not create a new import line in the generated code
				DefaultingFunc: "DefaultDaemonSet",
			},
			{
				ResourceName:       "PodDisruptionBudget",
				ImportAlias:        "policyv1beta1",
				ResourceImportPath: "k8s.io/api/policy/v1beta1",
				RequiresRecreate:   true,
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
				ResourceName: "ClusterRole",
				ImportAlias:  "rbacv1",
				// Don't specify ResourceImportPath so this block does not create a new import line in the generated code
			},
			{
				ResourceName: "Role",
				ImportAlias:  "rbacv1",
				// Don't specify ResourceImportPath so this block does not create a new import line in the generated code
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
			},
			{
				ResourceName:       "CronJob",
				ImportAlias:        "batchv1beta1",
				ResourceImportPath: "k8s.io/api/batch/v1beta1",
				DefaultingFunc:     "DefaultCronJob",
			},
			{
				ResourceName:       "MutatingWebhookConfiguration",
				ImportAlias:        "admissionregistrationv1beta1",
				ResourceImportPath: "k8s.io/api/admissionregistration/v1beta1",
			},
			{
				ResourceName: "ValidatingWebhookConfiguration",
				ImportAlias:  "admissionregistrationv1beta1",
				// Don't specify ResourceImportPath so this block does not create a new import line in the generated code
			},
			{
				ResourceName:       "APIService",
				ImportAlias:        "apiregistrationv1beta1",
				ResourceImportPath: "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1",
			},
			{
				ResourceName:       "Ingress",
				ResourceNamePlural: "Ingresses",
				ImportAlias:        "networkingv1beta1",
				ResourceImportPath: "k8s.io/api/networking/v1beta1",
			},
			{
				ResourceName:       "Seed",
				ImportAlias:        "kubermaticv1",
				ResourceImportPath: "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1",
			},
			{
				ResourceName:       "Certificate",
				ImportAlias:        "certmanagerv1alpha2",
				ResourceImportPath: "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2",
			},
			{
				ResourceName:       "ConstraintTemplate",
				ImportAlias:        "gatekeeperv1beta1",
				ResourceImportPath: "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1",
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
		"namedReconcileFunc": namedReconcileFunc,
	}
	reconcileAllTemplate = template.Must(template.New("").Funcs(reconcileAllTplFuncs).Funcs(sprig.TxtFuncMap()).Parse(`// This file is generated. DO NOT EDIT.
package reconciling

import (
	"fmt"
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
{{ range .Resources }}
{{- if .ResourceImportPath }}
	{{ .ImportAlias }} "{{ .ResourceImportPath }}"
{{- end }}
{{- end }}
)

{{ range .Resources }}
{{ namedReconcileFunc .ResourceName .ImportAlias .DefaultingFunc .RequiresRecreate .ResourceNamePlural }}
{{- end }}

`))
)

type reconcileFunctionData struct {
	ResourceName       string
	ResourceNamePlural string
	ResourceImportPath string
	ImportAlias        string
	// Optional: A defaulting func for the given object type
	// Must be defined inside the resources package
	DefaultingFunc string
	// Whether the resource must be recreated instead of updated. Required
	// e.G. for PDBs
	RequiresRecreate bool
}

func namedReconcileFunc(resourceName, importAlias, defaultingFunc string, requiresRecreate bool, plural string) (string, error) {
	if len(plural) == 0 {
		plural = fmt.Sprintf("%ss", resourceName)
	}

	b := &bytes.Buffer{}
	err := namedReconcileFunctionTemplate.Execute(b, struct {
		ResourceName       string
		ResourceNamePlural string
		ImportAlias        string
		DefaultingFunc     string
		RequiresRecreate   bool
	}{
		ResourceName:       resourceName,
		ResourceNamePlural: plural,
		ImportAlias:        importAlias,
		DefaultingFunc:     defaultingFunc,
		RequiresRecreate:   requiresRecreate,
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
)

var namedReconcileFunctionTemplate = template.Must(template.New("").Funcs(reconcileFunctionTplFuncs).Parse(`// {{ .ResourceName }}Creator defines an interface to create/update {{ .ResourceName }}s
type {{ .ResourceName }}Creator = func(existing *{{ .ImportAlias }}.{{ .ResourceName }}) (*{{ .ImportAlias }}.{{ .ResourceName }}, error)

// Named{{ .ResourceName }}CreatorGetter returns the name of the resource and the corresponding creator function
type Named{{ .ResourceName }}CreatorGetter = func() (name string, create {{ .ResourceName }}Creator)

// {{ .ResourceName }}ObjectWrapper adds a wrapper so the {{ .ResourceName }}Creator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func {{ .ResourceName }}ObjectWrapper(create {{ .ResourceName }}Creator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*{{ .ImportAlias }}.{{ .ResourceName }}))
		}
		return create(&{{ .ImportAlias }}.{{ .ResourceName }}{})
	}
}

// Reconcile{{ .ResourceNamePlural }} will create and update the {{ .ResourceNamePlural }} coming from the passed {{ .ResourceName }}Creator slice
func Reconcile{{ .ResourceNamePlural }}(ctx context.Context, namedGetters []Named{{ .ResourceName }}CreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
{{- if .DefaultingFunc }}
		create = {{ .DefaultingFunc }}(create)
{{- end }}
		createObject := {{ .ResourceName }}ObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &{{ .ImportAlias }}.{{ .ResourceName }}{}, {{ .RequiresRecreate}}); err != nil {
			return fmt.Errorf("failed to ensure {{ .ResourceName }} %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

`))
