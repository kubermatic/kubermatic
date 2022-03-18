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

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	e2eutils "k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	vsphereSecretPrefixName = "credentials-vsphere"

	ccmDeploymentName = "vsphere-cloud-controller-manager"
)

type VsphereClusterJig struct {
	CommonClusterJig

	log *zap.SugaredLogger

	Credentials VsphereCredentialsType
}

func NewClusterJigVsphere(seedClient ctrlruntimeclient.Client, version semver.Semver, seedDatacenter string, credentials VsphereCredentialsType) *VsphereClusterJig {
	return &VsphereClusterJig{
		CommonClusterJig: CommonClusterJig{
			name:           fmt.Sprintf("v%s", rand.String(9)),
			DatacenterName: seedDatacenter,
			Version:        version,
			SeedClient:     seedClient,
		},
		log:         e2eutils.DefaultLogger,
		Credentials: credentials,
	}
}

func (c *VsphereClusterJig) Setup() error {
	c.log.Debugw("Setting up new cluster", "name", c.Name)

	if err := c.generateAndCreateSecret(vsphereSecretPrefixName, c.Credentials.GenerateSecretData()); err != nil {
		return errors.Wrap(err, "failed to create credential secret")
	}
	c.log.Debugw("secret created", "name", fmt.Sprintf("%s-%s", vsphereSecretPrefixName, c.name))

	if err := c.generateAndCreateCluster(kubermaticv1.CloudSpec{
		DatacenterName: c.DatacenterName,
		VSphere: &kubermaticv1.VSphereCloudSpec{
			CredentialsReference: &types.GlobalSecretKeySelector{
				ObjectReference: corev1.ObjectReference{
					Name:      fmt.Sprintf("%s-%s", vsphereSecretPrefixName, c.name),
					Namespace: resources.KubermaticNamespace,
				},
			},
		},
	}); err != nil {
		return errors.Wrap(err, "failed to create user cluster")
	}
	c.log.Debugw("Cluster created", "name", c.Name)

	return c.waitForClusterControlPlaneReady()
}

func (c *VsphereClusterJig) CreateMachineDeployment(userClient ctrlruntimeclient.Client) error {
	if err := c.generateAndCreateMachineDeployment(userClient, c.Credentials.GenerateProviderSpec(c.name)); err != nil {
		return errors.Wrap(err, "failed to create machine deployment")
	}
	return nil
}

func (c *VsphereClusterJig) Cleanup(userClient ctrlruntimeclient.Client) error {
	return c.cleanUp(userClient)
}

func (c *VsphereClusterJig) CheckComponents(userClient ctrlruntimeclient.Client) (bool, error) {
	ctx := context.TODO()

	ccmDeploy := &appsv1.Deployment{}
	if err := c.SeedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: fmt.Sprintf("cluster-%s", c.name), Name: ccmDeploymentName}, ccmDeploy); err != nil {
		return false, errors.Wrapf(err, "failed to get %s deployment", ccmDeploymentName)
	}
	if ccmDeploy.Status.AvailableReplicas == 1 {
		return true, nil
	}

	return false, nil
}

func (c *VsphereClusterJig) Name() string {
	return c.name
}

func (c *VsphereClusterJig) Seed() ctrlruntimeclient.Client {
	return c.SeedClient
}

func (c *VsphereClusterJig) Log() *zap.SugaredLogger {
	return c.log
}

type VsphereCredentialsType struct {
	AuthURL    string
	Username   string
	Password   string
	Datacenter string
	Cluster    string
}

func (vc *VsphereCredentialsType) GenerateSecretData() map[string][]byte {
	return map[string][]byte{
		resources.VsphereUsername:                    []byte(vc.Username),
		resources.VspherePassword:                    []byte(vc.Password),
		resources.VsphereInfraManagementUserUsername: []byte(""),
		resources.VsphereInfraManagementUserPassword: []byte(""),
	}
}

func (vc *VsphereCredentialsType) GenerateProviderSpec(clustername string) []byte {
	cloudProviderSpec := fmt.Sprintf(`{"allowInsecure":false,"cluster":"%s","cpus":2,"datacenter":"%s","datastore":"HS-FreeNAS","datastoreCluster":"","diskSizeGB":10,"memoryMB":4096,"folder":"/%s/vm/e2e-tests/%s","templateVMName":"machine-controller-e2e-ubuntu"}`, vc.Cluster, vc.Datacenter, vc.Datacenter, clustername)
	return []byte(fmt.Sprintf(`{"cloudProvider":"vsphere","cloudProviderSpec":%s,"operatingSystem":"ubuntu","operatingSystemSpec":{"distUpgradeOnBoot":false}}`,
		cloudProviderSpec))
}
