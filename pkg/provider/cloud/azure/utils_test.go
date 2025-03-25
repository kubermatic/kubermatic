//go:build integration

/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package azure

import (
	"context"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type fakeClientMode string

const (
	testLocation = "westeurope"

	fakeClientModeOkay     fakeClientMode = "okay"
	fakeClientModeAuthFail fakeClientMode = "authfail"
)

func getFakeCredentials() (*Credentials, error) {
	// All of these UUIDs are meaningless and generated with `uuidgen`.
	return &Credentials{
		TenantID:       "f2ff9b16-5461-48a1-9b4f-a47b1acd6371",
		SubscriptionID: "39b60bc3-21c1-4eee-9940-c46e41fc18b1",
		ClientID:       "f40312ea-a69c-4f4a-9c21-2f1d890135ff",
		ClientSecret:   "f1c5b2df-9ed9-4d17-bb4a-8e8e0ff7cb9f",
	}, nil
}

// makeCluster returns a KKP Cluster object with the Azure cloud spec inserted.
func makeCluster(name string, cloudSpec *kubermaticv1.AzureCloudSpec, credentials *Credentials) *kubermaticv1.Cluster {
	spec := cloudSpec.DeepCopy()
	spec.TenantID = credentials.TenantID
	spec.SubscriptionID = credentials.SubscriptionID
	spec.ClientID = credentials.ClientID
	spec.ClientSecret = credentials.ClientSecret

	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.ClusterSpec{
			Cloud: kubermaticv1.CloudSpec{
				Azure: spec,
			},
		},
	}
}

func testClusterUpdater(cluster *kubermaticv1.Cluster) provider.ClusterUpdater {
	return func(_ context.Context, clusterName string, patcher func(*kubermaticv1.Cluster)) (*kubermaticv1.Cluster, error) {
		patcher(cluster)
		return cluster, nil
	}
}
