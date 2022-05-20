package machine

import (
	"context"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"testing"
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
