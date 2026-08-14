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

package machine

import (
	"crypto/x509"
	"slices"
	"testing"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
)

func TestValidatingWebhookConfigurationOperations(t *testing.T) {
	_, reconciler := ValidatingWebhookConfigurationReconciler(testCertificate(), "cluster-abcd")()
	configuration, err := reconciler(&admissionregistrationv1.ValidatingWebhookConfiguration{})
	if err != nil {
		t.Fatalf("reconcile validating webhook: %v", err)
	}
	if len(configuration.Webhooks) != 1 || len(configuration.Webhooks[0].Rules) != 1 {
		t.Fatalf("unexpected webhook configuration: %#v", configuration.Webhooks)
	}
	if got := configuration.Webhooks[0].Rules[0].Operations; !slices.Equal(got, []admissionregistrationv1.OperationType{admissionregistrationv1.Create}) {
		t.Fatalf("operations = %v, want [CREATE]", got)
	}
}

func TestAcceleratorMutatingWebhookConfiguration(t *testing.T) {
	name, reconciler := AcceleratorMutatingWebhookConfigurationReconciler(testCertificate(), "cluster-abcd")()
	if name != AcceleratorAdmissionWebhookName {
		t.Fatalf("name = %q, want %q", name, AcceleratorAdmissionWebhookName)
	}
	configuration, err := reconciler(&admissionregistrationv1.MutatingWebhookConfiguration{})
	if err != nil {
		t.Fatalf("reconcile mutating webhook: %v", err)
	}
	if len(configuration.Webhooks) != 1 {
		t.Fatalf("webhook count = %d, want 1", len(configuration.Webhooks))
	}
	hook := configuration.Webhooks[0]
	const expectedURL = "https://usercluster-webhook.cluster-abcd.svc.cluster.local.:6443/mutate-machine-accelerator-footprint"
	if hook.ClientConfig.URL == nil || *hook.ClientConfig.URL != expectedURL {
		if hook.ClientConfig.URL == nil {
			t.Fatal("URL is nil")
		}
		t.Fatalf("URL = %q, want %q", *hook.ClientConfig.URL, expectedURL)
	}
	if hook.FailurePolicy == nil || *hook.FailurePolicy != admissionregistrationv1.Fail {
		t.Fatalf("failurePolicy = %v, want Fail", hook.FailurePolicy)
	}
	if hook.ReinvocationPolicy == nil || *hook.ReinvocationPolicy != admissionregistrationv1.IfNeededReinvocationPolicy {
		t.Fatalf("reinvocationPolicy = %v, want IfNeeded", hook.ReinvocationPolicy)
	}
	if len(hook.Rules) != 1 || !slices.Equal(hook.Rules[0].Operations, []admissionregistrationv1.OperationType{admissionregistrationv1.Create}) {
		t.Fatalf("rules = %#v, want Machine CREATE", hook.Rules)
	}
}

func TestAcceleratorValidatingWebhookConfiguration(t *testing.T) {
	name, reconciler := AcceleratorValidatingWebhookConfigurationReconciler(testCertificate(), "cluster-abcd")()
	if name != AcceleratorAdmissionWebhookName {
		t.Fatalf("name = %q, want %q", name, AcceleratorAdmissionWebhookName)
	}
	configuration, err := reconciler(&admissionregistrationv1.ValidatingWebhookConfiguration{})
	if err != nil {
		t.Fatalf("reconcile validating webhook: %v", err)
	}
	if len(configuration.Webhooks) != 1 {
		t.Fatalf("webhook count = %d, want 1", len(configuration.Webhooks))
	}
	hook := configuration.Webhooks[0]
	const expectedURL = "https://usercluster-webhook.cluster-abcd.svc.cluster.local.:6443/validate-machine-accelerator-footprint"
	if hook.ClientConfig.URL == nil || *hook.ClientConfig.URL != expectedURL {
		if hook.ClientConfig.URL == nil {
			t.Fatal("URL is nil")
		}
		t.Fatalf("URL = %q, want %q", *hook.ClientConfig.URL, expectedURL)
	}
	if hook.FailurePolicy == nil || *hook.FailurePolicy != admissionregistrationv1.Fail {
		t.Fatalf("failurePolicy = %v, want Fail", hook.FailurePolicy)
	}
	wantOperations := []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update}
	if len(hook.Rules) != 1 || !slices.Equal(hook.Rules[0].Operations, wantOperations) {
		t.Fatalf("rules = %#v, want operations %v", hook.Rules, wantOperations)
	}
}

func testCertificate() *x509.Certificate {
	return &x509.Certificate{Raw: []byte("test-ca")}
}
