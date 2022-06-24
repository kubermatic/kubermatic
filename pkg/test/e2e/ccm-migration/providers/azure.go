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

package providers

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	utilcluster "k8c.io/kubermatic/v2/pkg/util/cluster"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	azureSecretPrefixName  = "credentials-azure"
	azureNodeDaemonSetName = "cloud-node-manager"
	azureCCMDeploymentName = "azure-cloud-controller-manager"
)

func NewClusterJigAzure(seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger, version semver.Semver, seedDatacenter string, credentials AzureCredentialsType) *AzureClusterJig {
	return &AzureClusterJig{
		CommonClusterJig: CommonClusterJig{
			name:           utilcluster.MakeClusterName(),
			DatacenterName: seedDatacenter,
			Version:        version,
			SeedClient:     seedClient,
			log:            log,
		},
		Credentials: credentials,
	}
}

var (
	_ ClusterJigInterface = &AzureClusterJig{}
)

type AzureClusterJig struct {
	CommonClusterJig

	Credentials AzureCredentialsType
	cluster     kubermaticv1.Cluster
}

func (c *AzureClusterJig) Setup(ctx context.Context) error {
	c.log.Debugw("Setting up new cluster", "name", c.name)

	projectID := rand.String(10)
	if err := c.generateAndCreateProject(ctx, projectID); err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}
	c.log.Debugw("project created", "name", projectID)

	if err := c.generateAndCreateSecret(ctx, azureSecretPrefixName, c.Credentials.GenerateSecretData()); err != nil {
		return fmt.Errorf("failed to create credential secret: %w", err)
	}
	c.log.Debugw("secret created", "name", fmt.Sprintf("%s-%s", azureSecretPrefixName, c.name))

	if err := c.generateAndCreateCluster(ctx, kubermaticv1.CloudSpec{
		DatacenterName: c.DatacenterName,
		Azure: &kubermaticv1.AzureCloudSpec{
			CredentialsReference: &types.GlobalSecretKeySelector{
				ObjectReference: corev1.ObjectReference{
					Name:      fmt.Sprintf("%s-%s", azureSecretPrefixName, c.name),
					Namespace: resources.KubermaticNamespace,
				},
			},
		},
	}, projectID); err != nil {
		return fmt.Errorf("failed to create user cluster: %w", err)
	}
	c.log.Debugw("Cluster created", "name", c.name)

	if err := c.waitForClusterControlPlaneReady(ctx); err != nil {
		return fmt.Errorf("failed to wait for cluster control plane: %w", err)
	}

	if err := c.SeedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: c.name}, &c.cluster); err != nil {
		return fmt.Errorf("failed to get user cluster: %w", err)
	}

	return nil
}

func (c *AzureClusterJig) CreateMachineDeployment(ctx context.Context, userClient ctrlruntimeclient.Client) error {
	if err := c.generateAndCreateMachineDeployment(ctx, userClient, c.Credentials.GenerateProviderSpec(c.cluster.Spec.Cloud.Azure)); err != nil {
		return fmt.Errorf("failed to create machine deployment: %w", err)
	}
	return nil
}

func (c *AzureClusterJig) Cleanup(ctx context.Context, userClient ctrlruntimeclient.Client) error {
	return c.cleanUp(ctx, userClient)
}

func (c *AzureClusterJig) CheckComponents(ctx context.Context, userClient ctrlruntimeclient.Client) (bool, error) {
	ccmDeploy := &appsv1.Deployment{}
	if err := c.SeedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: fmt.Sprintf("cluster-%s", c.name), Name: azureCCMDeploymentName}, ccmDeploy); err != nil {
		return false, fmt.Errorf("failed to get %s deployment: %w", azureCCMDeploymentName, err)
	}
	if ccmDeploy.Status.AvailableReplicas == 1 {
		return true, nil
	}

	nodeDaemonSet := &appsv1.DaemonSet{}
	if err := userClient.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: metav1.NamespaceSystem, Name: azureNodeDaemonSetName}, nodeDaemonSet); err != nil {
		return false, fmt.Errorf("failed to get %s daemonset: %w", azureNodeDaemonSetName, err)
	}

	if nodeDaemonSet.Status.NumberReady == nodeDaemonSet.Status.DesiredNumberScheduled {
		return true, nil
	}

	return false, nil
}

func (c *AzureClusterJig) Name() string {
	return c.name
}

func (c *AzureClusterJig) Seed() ctrlruntimeclient.Client {
	return c.SeedClient
}

func (c *AzureClusterJig) Log() *zap.SugaredLogger {
	return c.log
}

type AzureCredentialsType struct {
	resources.AzureCredentials
}

func (c *AzureCredentialsType) GenerateSecretData() map[string][]byte {
	return map[string][]byte{
		"clientID":       []byte(c.ClientID),
		"clientSecret":   []byte(c.ClientSecret),
		"subscriptionID": []byte(c.SubscriptionID),
		"tenantID":       []byte(c.TenantID),
	}
}

func (c *AzureCredentialsType) GenerateProviderSpec(spec *kubermaticv1.AzureCloudSpec) []byte {
	return []byte(fmt.Sprintf(`{"cloudProvider": "azure", "cloudProviderSpec": {"tenantID": "%s", "clientID": "%s", "clientSecret": "%s", "subscriptionID": "%s", "location": "westeurope", "vmSize": "Standard_B1ms", "resourceGroup": "%s", "vnetName": "%s", "subnetName": "%s", "routeTableName": "%s", "securityGroupName": "%s"}, "operatingSystem": "ubuntu", "operatingSystemSpec": {"distUpgradeOnBoot": false, "disableAutoUpdate": true}}`,
		c.TenantID,
		c.ClientID,
		c.ClientSecret,
		c.SubscriptionID,
		spec.ResourceGroup,
		spec.VNetName,
		spec.SubnetName,
		spec.RouteTableName,
		spec.SecurityGroup,
	))
}
