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

package resources

import (
	"context"
	"crypto/x509"
	"slices"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/machine"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestAcceleratorFootprintAdmissionActive(t *testing.T) {
	testCases := []struct {
		name     string
		provider string
		enabled  bool
		objects  []ctrlruntimeclient.Object
		want     bool
	}{
		{
			name:     "never enabled",
			provider: string(kubermaticv1.KubevirtCloudProvider),
		},
		{
			name:     "enabled for this cluster",
			provider: string(kubermaticv1.KubevirtCloudProvider),
			enabled:  true,
			want:     true,
		},
		{
			name:     "mutating configuration keeps activation sticky",
			provider: string(kubermaticv1.KubevirtCloudProvider),
			objects: []ctrlruntimeclient.Object{&admissionregistrationv1.MutatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{Name: machine.AcceleratorAdmissionWebhookName},
			}},
			want: true,
		},
		{
			name:     "validating configuration keeps activation sticky",
			provider: string(kubermaticv1.KubevirtCloudProvider),
			objects: []ctrlruntimeclient.Object{&admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{Name: machine.AcceleratorAdmissionWebhookName},
			}},
			want: true,
		},
		{
			name:     "non KubeVirt clusters remain inactive",
			provider: string(kubermaticv1.AWSCloudProvider),
			enabled:  true,
			objects: []ctrlruntimeclient.Object{&admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{Name: machine.AcceleratorAdmissionWebhookName},
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			if err := admissionregistrationv1.AddToScheme(scheme); err != nil {
				t.Fatalf("add admissionregistration API to scheme: %v", err)
			}
			client := ctrlruntimefakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(tc.objects...).Build()
			r := &reconciler{Client: client, kubeVirtAcceleratorQuota: tc.enabled}

			got, err := r.acceleratorFootprintAdmissionActive(context.Background(), reconcileData{cloudProviderName: tc.provider})
			if err != nil {
				t.Fatalf("acceleratorFootprintAdmissionActive() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("acceleratorFootprintAdmissionActive() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestReconcileAcceleratorFootprintAdmission(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := admissionregistrationv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add admissionregistration API to scheme: %v", err)
	}
	client := ctrlruntimefakeclient.NewClientBuilder().WithScheme(scheme).Build()
	r := &reconciler{
		Client:                   client,
		namespace:                "cluster-abcd",
		kubeVirtAcceleratorQuota: true,
	}
	data := reconcileData{
		caCert:            &triple.KeyPair{Cert: &x509.Certificate{Raw: []byte("test-ca")}},
		cloudProviderName: string(kubermaticv1.KubevirtCloudProvider),
	}

	if err := r.reconcileAcceleratorFootprintAdmission(context.Background(), data); err != nil {
		t.Fatalf("reconcileAcceleratorFootprintAdmission() error = %v", err)
	}
	assertAcceleratorAdmissionConfigurations(t, client)

	if err := client.Delete(context.Background(), &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: machine.AcceleratorAdmissionWebhookName},
	}); err != nil {
		t.Fatalf("delete mutating configuration: %v", err)
	}
	validation := &admissionregistrationv1.ValidatingWebhookConfiguration{}
	if err := client.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: machine.AcceleratorAdmissionWebhookName}, validation); err != nil {
		t.Fatalf("get validating configuration: %v", err)
	}
	validation.Webhooks[0].FailurePolicy = ptr.To(admissionregistrationv1.Ignore)
	if err := client.Update(context.Background(), validation); err != nil {
		t.Fatalf("drift validating configuration: %v", err)
	}
	if err := r.reconcileAcceleratorFootprintAdmission(context.Background(), data); err != nil {
		t.Fatalf("repair reconcileAcceleratorFootprintAdmission() error = %v", err)
	}
	assertAcceleratorAdmissionConfigurations(t, client)
}

func assertAcceleratorAdmissionConfigurations(t *testing.T, client ctrlruntimeclient.Client) {
	t.Helper()

	mutation := &admissionregistrationv1.MutatingWebhookConfiguration{}
	if err := client.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: machine.AcceleratorAdmissionWebhookName}, mutation); err != nil {
		t.Fatalf("get mutating configuration: %v", err)
	}
	if len(mutation.Webhooks) != 1 || len(mutation.Webhooks[0].Rules) != 1 ||
		!slices.Equal(mutation.Webhooks[0].Rules[0].Operations, []admissionregistrationv1.OperationType{admissionregistrationv1.Create}) {
		t.Fatalf("unexpected mutating configuration: %#v", mutation.Webhooks)
	}

	validation := &admissionregistrationv1.ValidatingWebhookConfiguration{}
	if err := client.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: machine.AcceleratorAdmissionWebhookName}, validation); err != nil {
		t.Fatalf("get validating configuration: %v", err)
	}
	wantOperations := []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update}
	if len(validation.Webhooks) != 1 || len(validation.Webhooks[0].Rules) != 1 ||
		!slices.Equal(validation.Webhooks[0].Rules[0].Operations, wantOperations) ||
		validation.Webhooks[0].FailurePolicy == nil || *validation.Webhooks[0].FailurePolicy != admissionregistrationv1.Fail {
		t.Fatalf("unexpected validating configuration: %#v", validation.Webhooks)
	}
}
