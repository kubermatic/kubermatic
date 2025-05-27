/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package applications

import (
	"context"
	"fmt"
	"testing"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultNamespace = "default"
)

func TestApplicationManager_applyNamespaceWithCreateNs(t *testing.T) {
	testCases := []struct {
		name          string
		userClient    ctrlruntimeclient.Client
		namespaceSpec appskubermaticv1.AppNamespaceSpec
	}{
		{
			name: "scenario 1: when Namespace.create=true and no labels or annotations are defined then namespace should be created without labels or annotations",
			userClient: fake.
				NewClientBuilder().
				Build(),
			namespaceSpec: appskubermaticv1.AppNamespaceSpec{
				Name:        "foo",
				Create:      true,
				Labels:      nil,
				Annotations: nil,
			},
		},
		{
			name: "scenario 2: when Namespace.create=true, labels field is defined and annotations field nil then namespace should be created with labels",
			userClient: fake.
				NewClientBuilder().
				Build(),
			namespaceSpec: appskubermaticv1.AppNamespaceSpec{
				Name:        "foo",
				Create:      true,
				Labels:      map[string]string{"label-1": "value-1", "label-2": "value-2"},
				Annotations: nil,
			},
		},
		{
			name: "scenario 3: when Namespace.create=true, labels field is nil and annotations field is defined then namespace should be created with annotations",
			userClient: fake.
				NewClientBuilder().
				Build(),
			namespaceSpec: appskubermaticv1.AppNamespaceSpec{
				Name:        "foo",
				Create:      true,
				Labels:      nil,
				Annotations: map[string]string{"annot-1": "value-1", "annot-2": "value-2"},
			},
		},
		{
			name: "scenario 4: when Namespace.create=true, labels and annotations are defined then namespace should be created with labels and annotations",
			userClient: fake.
				NewClientBuilder().
				Build(),
			namespaceSpec: appskubermaticv1.AppNamespaceSpec{
				Name:        "foo",
				Create:      true,
				Labels:      map[string]string{"label-1": "value-1", "label-2": "value-2"},
				Annotations: map[string]string{"annot-1": "value-1", "annot-2": "value-2"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			app := genApplicationInstallation(tc.namespaceSpec)
			appManager := &ApplicationManager{}
			if err := appManager.reconcileNamespace(ctx, kubermaticlog.Logger, app, tc.userClient); err != nil {
				t.Errorf("unexpected error when calling 'appManager.reconcileNamespace(...)': %v", err)
			}

			ns := &corev1.Namespace{}
			if err := tc.userClient.Get(ctx, types.NamespacedName{Name: tc.namespaceSpec.Name}, ns); err != nil {
				t.Errorf("failed to get created namespace: %v", err)
			}

			if err := contains(ns.Labels, tc.namespaceSpec.Labels); err != nil {
				t.Errorf("namespace does not contains expected labels: %v", err)
			}
			if err := contains(ns.Annotations, tc.namespaceSpec.Annotations); err != nil {
				t.Errorf("namespace does not contains expected annotations: %v", err)
			}
		})
	}
}

func TestApplicationManager_applyNamespaceDoNotCreateNsWhenCreateNamespaceFlagIsFalse(t *testing.T) {
	ctx := context.Background()
	userClient := fake.
		NewClientBuilder().
		Build()

	namespaceSpec := appskubermaticv1.AppNamespaceSpec{
		Name:        "foo",
		Create:      false,
		Labels:      nil,
		Annotations: nil,
	}

	app := genApplicationInstallation(namespaceSpec)
	appManager := &ApplicationManager{}
	if err := appManager.reconcileNamespace(ctx, kubermaticlog.Logger, app, userClient); err != nil {
		t.Errorf("unexpected error when calling 'appManager.reconcileNamespace(...)': %v", err)
	}

	ns := &corev1.Namespace{}
	err := userClient.Get(ctx, types.NamespacedName{Name: namespaceSpec.Name}, ns)
	if err == nil {
		t.Error("namespace should not have been created")
	}
	if !apierrors.IsNotFound(err) {
		t.Errorf("can not check that namespace has not been created: %v", err)
	}
}

func TestApplicationManager_applyNamespaceDoNotSetLabelsAndAnnotationWhenCreateNamespaceFlagIsFalse(t *testing.T) {
	ctx := context.Background()
	nsName := "foo"
	userClient := fake.
		NewClientBuilder().
		WithObjects(genNamespace(nsName), genNamespace(defaultNamespace)).
		Build()

	namespaceSpec := appskubermaticv1.AppNamespaceSpec{
		Name:        nsName,
		Create:      false,
		Labels:      nil,
		Annotations: nil,
	}

	app := genApplicationInstallation(namespaceSpec)
	appManager := &ApplicationManager{}
	if err := appManager.reconcileNamespace(ctx, kubermaticlog.Logger, app, userClient); err != nil {
		t.Errorf("unexpected error when calling 'appManager.reconcileNamespace(...)': %v", err)
	}

	ns := &corev1.Namespace{}
	if err := userClient.Get(ctx, types.NamespacedName{Name: nsName}, ns); err != nil && !apierrors.IsNotFound(err) {
		t.Errorf("failed to get manually created namespace: %v", err)
	}

	if ns.Labels != nil {
		t.Errorf("labels should not have been set. actual=%v", ns.Labels)
	}
	if ns.Annotations != nil {
		t.Errorf("Annotations should not have been set. actual=%v", ns.Labels)
	}
}

// contains returns an error if actual does not contain expected. If actual and expected are nil, no error is returned.
func contains(actual map[string]string, expected map[string]string) error {
	if expected == nil && actual != nil {
		return fmt.Errorf("expect '%v' to be nil but was not", actual)
	}
	if expected != nil && actual == nil {
		return fmt.Errorf("actual is nil but should contains '%v'", expected)
	}

	missingElements := make(map[string]string, len(expected))
	for k, v := range expected {
		missingElements[k] = v
	}

	for k, v := range actual {
		if expectedValue, found := expected[k]; found && v == expectedValue {
			delete(missingElements, k)
		}
	}
	if len(missingElements) != 0 {
		return fmt.Errorf("expect '%v' to contains '%v' but '%v' is missing", expected, actual, missingElements)
	}
	return nil
}

func genApplicationInstallation(namespaceSpec appskubermaticv1.AppNamespaceSpec) *appskubermaticv1.ApplicationInstallation {
	return &appskubermaticv1.ApplicationInstallation{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "app-",
			Namespace:    defaultNamespace,
		},
		Spec: appskubermaticv1.ApplicationInstallationSpec{
			Namespace: &namespaceSpec,
			ApplicationRef: appskubermaticv1.ApplicationRef{
				Name:    "applicationDef1",
				Version: "1.0.0",
			},
		},
		Status: appskubermaticv1.ApplicationInstallationStatus{},
	}
}

func genNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}
