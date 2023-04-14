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

package monitoringcontroller

import (
	"testing"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v3/pkg/test"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	TestDC = "regular-do1"
)

func newTestReconciler(t *testing.T, objects []ctrlruntimeclient.Object) *Reconciler {
	dynamicClient := ctrlruntimefakeclient.
		NewClientBuilder().
		WithObjects(objects...).
		WithObjects(
			&kubermaticv1.Datacenter{
				ObjectMeta: metav1.ObjectMeta{
					Name: "us-central1-byo",
				},
				Spec: kubermaticv1.DatacenterSpec{
					Provider: kubermaticv1.DatacenterProviderSpec{
						ProviderName: kubermaticv1.CloudProviderBringYourOwn,
						BringYourOwn: &kubermaticv1.DatacenterSpecBringYourOwn{},
					},
				},
			},
			&kubermaticv1.Datacenter{
				ObjectMeta: metav1.ObjectMeta{
					Name: "private-do1",
				},
				Spec: kubermaticv1.DatacenterSpec{
					Provider: kubermaticv1.DatacenterProviderSpec{
						ProviderName: kubermaticv1.CloudProviderDigitalocean,
						Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{
							Region: "ams2",
						},
					},
				},
			},
			&kubermaticv1.Datacenter{
				ObjectMeta: metav1.ObjectMeta{
					Name: TestDC,
				},
				Spec: kubermaticv1.DatacenterSpec{
					Provider: kubermaticv1.DatacenterProviderSpec{
						ProviderName: kubermaticv1.CloudProviderDigitalocean,
						Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{
							Region: "ams2",
						},
					},
				},
			},
		).
		Build()

	datacenterGetter, err := kubernetes.DatacenterGetterFactory(dynamicClient)
	if err != nil {
		t.Fatal(err)
	}

	reconciler := &Reconciler{
		Client:           dynamicClient,
		datacenterGetter: datacenterGetter,
		configGetter: test.NewConfigGetter(&kubermaticv1.KubermaticConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubermatic",
				Namespace: "kubermatic",
			},
		}),
		nodeAccessNetwork:    "192.0.2.0/24",
		dockerPullConfigJSON: []byte{},
	}

	return reconciler
}
