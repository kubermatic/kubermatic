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
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/uuid"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type fakeClientMode string

const (
	testLocation = "westeurope"

	fakeClientModeOkay     fakeClientMode = "okay"
	fakeClientModeAuthFail fakeClientMode = "authfail"
)

func getFakeCredentials() (*Credentials, error) {
	tenantID, err := uuid.UUID()
	if err != nil {
		return nil, err
	}

	subscriptionID, err := uuid.UUID()
	if err != nil {
		return nil, err
	}

	clientID, err := uuid.UUID()
	if err != nil {
		return nil, err
	}

	clientSecret, err := uuid.UUID()
	if err != nil {
		return nil, err
	}

	return &Credentials{
		TenantID:       tenantID,
		SubscriptionID: subscriptionID,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
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
	return func(clusterName string, patcher func(*kubermaticv1.Cluster)) (*kubermaticv1.Cluster, error) {
		patcher(cluster)
		return cluster, nil
	}
}
