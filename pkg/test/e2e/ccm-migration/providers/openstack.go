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

package providers

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	types2 "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	utilcluster "k8c.io/kubermatic/v2/pkg/util/cluster"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	osSecretPrefixName  = "credentials-openstack"
	osCCMDeploymentName = "openstack-cloud-controller-manager"
)

type OpenstackClusterJig struct {
	CommonClusterJig

	Credentials OpenstackCredentialsType
}

func NewClusterJigOpenstack(seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger, version semver.Semver, seedDatacenter string, credentials OpenstackCredentialsType) *OpenstackClusterJig {
	return &OpenstackClusterJig{
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

func (c *OpenstackClusterJig) Setup(ctx context.Context) error {
	c.log.Debugw("Setting up new cluster", "name", c.name)

	projectID := rand.String(10)
	if err := c.generateAndCreateProject(ctx, projectID); err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}
	c.log.Debugw("project created", "name", projectID)

	if err := c.generateAndCreateSecret(ctx, osSecretPrefixName, c.Credentials.GenerateSecretData()); err != nil {
		return fmt.Errorf("failed to create credential secret: %w", err)
	}
	c.log.Debugw("secret created", "name", fmt.Sprintf("%s-%s", osSecretPrefixName, c.name))

	if err := c.generateAndCreateCluster(ctx, kubermaticv1.CloudSpec{
		DatacenterName: c.DatacenterName,
		Openstack: &kubermaticv1.OpenstackCloudSpec{
			FloatingIPPool: c.Credentials.FloatingIPPool,
			CredentialsReference: &types2.GlobalSecretKeySelector{
				ObjectReference: corev1.ObjectReference{
					Name:      fmt.Sprintf("%s-%s", osSecretPrefixName, c.name),
					Namespace: resources.KubermaticNamespace,
				},
			},
		},
	}, projectID); err != nil {
		return fmt.Errorf("failed to create user cluster: %w", err)
	}
	c.log.Debugw("Cluster created", "name", c.Name())

	return c.waitForClusterControlPlaneReady(ctx)
}

func (c *OpenstackClusterJig) CreateMachineDeployment(ctx context.Context, userClient ctrlruntimeclient.Client) error {
	if err := c.generateAndCreateMachineDeployment(ctx, userClient, c.Credentials.GenerateProviderSpec()); err != nil {
		return fmt.Errorf("failed to create machine deployment: %w", err)
	}
	return nil
}

func (c *OpenstackClusterJig) Cleanup(ctx context.Context, userClient ctrlruntimeclient.Client) error {
	return c.cleanUp(ctx, userClient)
}

func (c *OpenstackClusterJig) CheckComponents(ctx context.Context, userClient ctrlruntimeclient.Client) (bool, error) {
	ccmDeploy := &appsv1.Deployment{}
	if err := c.SeedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: fmt.Sprintf("cluster-%s", c.name), Name: osCCMDeploymentName}, ccmDeploy); err != nil {
		return false, fmt.Errorf("failed to get %s deployment: %w", osCCMDeploymentName, err)
	}
	if ccmDeploy.Status.AvailableReplicas == 1 {
		return true, nil
	}

	return false, nil
}

func (c *OpenstackClusterJig) Name() string {
	return c.name
}

func (c *OpenstackClusterJig) Seed() ctrlruntimeclient.Client {
	return c.SeedClient
}

func (c *OpenstackClusterJig) Log() *zap.SugaredLogger {
	return c.log
}

type OpenstackCredentialsType struct {
	AuthURL        string
	Username       string
	Password       string
	Tenant         string
	Domain         string
	Region         string
	FloatingIPPool string
	Network        string
	Datacenter     string
}

func (osc *OpenstackCredentialsType) GenerateSecretData() map[string][]byte {
	return map[string][]byte{
		resources.OpenstackUsername:                    []byte(osc.Username),
		resources.OpenstackPassword:                    []byte(osc.Password),
		resources.OpenstackTenant:                      []byte(osc.Tenant),
		resources.OpenstackTenantID:                    []byte(""),
		resources.OpenstackDomain:                      []byte(osc.Domain),
		resources.OpenstackApplicationCredentialID:     []byte(""),
		resources.OpenstackApplicationCredentialSecret: []byte(""),
		resources.OpenstackToken:                       []byte(""),
	}
}

func (osc *OpenstackCredentialsType) GenerateProviderSpec() []byte {
	return []byte(fmt.Sprintf(`{"cloudProvider": "openstack","cloudProviderSpec": {"identityEndpoint": "%s","Username": "%s","Password": "%s", "tenantName": "%s", "Region": "%s", "domainName": "%s", "FloatingIPPool": "%s", "Network": "%s", "image": "machine-controller-e2e-ubuntu", "flavor": "m1.small"},"operatingSystem": "ubuntu","operatingSystemSpec":{"distUpgradeOnBoot": false,"disableAutoUpdate": true}}`,
		osc.AuthURL,
		osc.Username,
		osc.Password,
		osc.Tenant,
		osc.Region,
		osc.Domain,
		osc.FloatingIPPool,
		osc.Network))
}
