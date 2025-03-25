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

package monitoring

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TestDC = "regular-do1"
)

func newTestReconciler(_ *testing.T, objects []ctrlruntimeclient.Object) *Reconciler {
	dynamicClient := fake.
		NewClientBuilder().
		WithObjects(objects...).
		Build()

	reconciler := &Reconciler{
		Client:               dynamicClient,
		seedGetter:           seed,
		configGetter:         kubermaticConfiguration,
		nodeAccessNetwork:    "192.0.2.0/24",
		dockerPullConfigJSON: []byte{},
		features:             Features{},
	}

	return reconciler
}

func seed() (*kubermaticv1.Seed, error) {
	return &kubermaticv1.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "us-central1",
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.SeedSpec{
			Datacenters: map[string]kubermaticv1.Datacenter{
				"us-central1-byo": {
					Location: "us-central",
					Country:  "US",
					Spec: kubermaticv1.DatacenterSpec{
						BringYourOwn: &kubermaticv1.DatacenterSpecBringYourOwn{},
					},
				},
				"us-central1-baremetal": {
					Location: "us-central",
					Country:  "US",
					Spec: kubermaticv1.DatacenterSpec{
						Baremetal: &kubermaticv1.DatacenterSpecBaremetal{},
					},
				},
				"us-central1-edge": {
					Location: "us-central",
					Country:  "US",
					Spec: kubermaticv1.DatacenterSpec{
						Edge: &kubermaticv1.DatacenterSpecEdge{},
					},
				},
				"private-do1": {
					Location: "US ",
					Country:  "NL",
					Spec: kubermaticv1.DatacenterSpec{
						Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{
							Region: "ams2",
						},
					},
				},
				"regular-do1": {
					Location: "Amsterdam",
					Country:  "NL",
					Spec: kubermaticv1.DatacenterSpec{
						Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{
							Region: "ams2",
						},
					},
				},
			},
		},
	}, nil
}

func kubermaticConfiguration(_ context.Context) (*kubermaticv1.KubermaticConfiguration, error) {
	return &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubermatic",
			Namespace: "kubermatic",
		},
	}, nil
}
