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
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/machine"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestAcceleratorFootprintAdmissionActive(t *testing.T) {
	testCases := []struct {
		name     string
		provider string
		gate     bool
		objects  []ctrlruntimeclient.Object
		want     bool
	}{
		{
			name:     "never enabled",
			provider: string(kubermaticv1.KubevirtCloudProvider),
		},
		{
			name:     "enabled by feature gate",
			provider: string(kubermaticv1.KubevirtCloudProvider),
			gate:     true,
			want:     true,
		},
		{
			name:     "mutating configuration keeps activation sticky",
			provider: string(kubermaticv1.KubevirtCloudProvider),
			objects: []ctrlruntimeclient.Object{&admissionregistrationv1.MutatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{Name: machine.AcceleratorMutatingWebhookConfigurationName},
			}},
			want: true,
		},
		{
			name:     "validating configuration keeps activation sticky",
			provider: string(kubermaticv1.KubevirtCloudProvider),
			objects: []ctrlruntimeclient.Object{&admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{Name: machine.AcceleratorValidatingWebhookConfigurationName},
			}},
			want: true,
		},
		{
			name:     "non KubeVirt clusters remain inactive",
			provider: string(kubermaticv1.AWSCloudProvider),
			gate:     true,
			objects: []ctrlruntimeclient.Object{&admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{Name: machine.AcceleratorValidatingWebhookConfigurationName},
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
			r := &reconciler{Client: client, kubeVirtAcceleratorQuota: tc.gate}

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
