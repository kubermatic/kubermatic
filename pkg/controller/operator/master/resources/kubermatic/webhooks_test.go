/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package kubermatic

import (
	"context"
	"strings"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestResourceQuotaValidatingWebhookOperations(t *testing.T) {
	const namespace = "kubermatic"
	client := fake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.WebhookServingCASecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			resources.CACertSecretKey: []byte("test-ca"),
		},
	}).Build()
	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
	}

	_, reconciler := ResourceQuotaValidatingWebhookConfigurationReconciler(context.Background(), cfg, client)()
	configuration, err := reconciler(&admissionregistrationv1.ValidatingWebhookConfiguration{})
	if err != nil {
		t.Fatalf("failed to reconcile ResourceQuota validating webhook: %v", err)
	}
	if len(configuration.Webhooks) != 1 || len(configuration.Webhooks[0].Rules) != 1 {
		t.Fatalf("expected one ResourceQuota webhook with one rule, got %#v", configuration.Webhooks)
	}

	operations := sets.New(configuration.Webhooks[0].Rules[0].Operations...)
	for _, expected := range []admissionregistrationv1.OperationType{
		admissionregistrationv1.Create,
		admissionregistrationv1.Update,
	} {
		if !operations.Has(expected) {
			t.Errorf("expected ResourceQuota webhook to handle %s", expected)
		}
	}
	if operations.Len() != 2 {
		t.Errorf("expected exactly CREATE and UPDATE operations, got %v", sets.List(operations))
	}
}

func TestResourceQuotaAcceleratorAccountingWebhook(t *testing.T) {
	const namespace = "kubermatic"
	client := fake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: common.WebhookServingCASecretName, Namespace: namespace},
		Data:       map[string][]byte{resources.CACertSecretKey: []byte("test-ca")},
	}).Build()
	cfg := &kubermaticv1.KubermaticConfiguration{ObjectMeta: metav1.ObjectMeta{Namespace: namespace}}

	name, reconciler := ResourceQuotaAcceleratorAccountingValidatingWebhookConfigurationReconciler(context.Background(), cfg, client)()
	if name != common.ResourceQuotaAcceleratorAccountingAdmissionWebhookName {
		t.Fatalf("name = %q, want %q", name, common.ResourceQuotaAcceleratorAccountingAdmissionWebhookName)
	}
	configuration, err := reconciler(&admissionregistrationv1.ValidatingWebhookConfiguration{})
	if err != nil {
		t.Fatalf("failed to reconcile accelerator-accounting webhook: %v", err)
	}
	if len(configuration.Webhooks) != 1 || len(configuration.Webhooks[0].MatchConditions) != 1 {
		t.Fatalf("expected one webhook with one match condition, got %#v", configuration.Webhooks)
	}
	hook := configuration.Webhooks[0]
	if hook.ClientConfig.Service == nil || hook.ClientConfig.Service.Path == nil || *hook.ClientConfig.Service.Path != resources.AcceleratorAccountingWebhookPath {
		t.Fatalf("service = %#v, want dedicated accelerator-accounting path", hook.ClientConfig.Service)
	}
	if hook.FailurePolicy == nil || *hook.FailurePolicy != admissionregistrationv1.Fail {
		t.Fatalf("failurePolicy = %v, want Fail", hook.FailurePolicy)
	}
	if len(hook.Rules) != 1 {
		t.Fatalf("expected one accelerator-accounting webhook rule, got %#v", hook.Rules)
	}
	operations := sets.New(hook.Rules[0].Operations...)
	for _, expected := range []admissionregistrationv1.OperationType{
		admissionregistrationv1.Create,
		admissionregistrationv1.Update,
		admissionregistrationv1.Delete,
	} {
		if !operations.Has(expected) {
			t.Errorf("expected accelerator-accounting webhook to handle %s", expected)
		}
	}
	if operations.Len() != 3 {
		t.Errorf("expected exactly CREATE, UPDATE, and DELETE operations, got %v", sets.List(operations))
	}
	expression := hook.MatchConditions[0].Expression
	for _, operand := range []string{"object.metadata.annotations", "oldObject.metadata.annotations", resources.AcceleratorAccountingEnabledAnnotation} {
		if !strings.Contains(expression, operand) {
			t.Errorf("match condition %q does not contain %q", expression, operand)
		}
	}
}
