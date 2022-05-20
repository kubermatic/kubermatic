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

package machine

import (
	"context"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
)

func TestGetProviderConfig(t *testing.T) {
	tests := []struct {
		name        string
		nd          *apiv1.NodeDeployment
		dc          *kubermaticv1.Datacenter
		expectError bool
	}{
		{
			name: "mismatched cloud provider",
			nd: &apiv1.NodeDeployment{
				ObjectMeta: apiv1.ObjectMeta{},
				Spec: apiv1.NodeDeploymentSpec{
					Replicas: 0,
					Template: apiv1.NodeSpec{
						Cloud: apiv1.NodeCloudSpec{
							Digitalocean: nil,
							AWS:          &apiv1.AWSNodeSpec{},
						},
						OperatingSystem: apiv1.OperatingSystemSpec{
							Ubuntu: &apiv1.UbuntuSpec{},
						},
					},
				},
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					Azure: &kubermaticv1.DatacenterSpecAzure{},
				},
			},
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var c kubermaticv1.Cluster
			c.Spec.Cloud.Azure = &kubermaticv1.AzureCloudSpec{}
			_, err := getProviderConfig(&c, test.nd, test.dc, nil,
				resources.NewCredentialsData(context.Background(), &c, nil))
			if (err != nil) != test.expectError {
				t.Fatalf("expected error: %t, got: %v", test.expectError, err)
			}
		})
	}
}
