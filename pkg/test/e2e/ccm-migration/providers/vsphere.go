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
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	vspheretypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vsphere/types"
	"github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	utilcluster "k8c.io/kubermatic/v2/pkg/util/cluster"
	"k8c.io/operating-system-manager/pkg/providerconfig/ubuntu"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	vsphereSecretPrefixName  = "credentials-vsphere"
	vsphereCCMDeploymentName = "vsphere-cloud-controller-manager"
)

type VsphereClusterJig struct {
	CommonClusterJig

	Credentials VsphereCredentialsType
}

func NewClusterJigVsphere(seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger, version semver.Semver, seedDatacenter string, credentials VsphereCredentialsType) *VsphereClusterJig {
	return &VsphereClusterJig{
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

func (c *VsphereClusterJig) Setup(ctx context.Context) error {
	c.log.Debugw("Setting up new cluster", "name", c.name)

	projectID := rand.String(10)
	if err := c.generateAndCreateProject(ctx, projectID); err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}
	c.log.Debugw("project created", "name", projectID)

	datacenter, err := c.getDatacenter(ctx, c.DatacenterName)
	if err != nil {
		return fmt.Errorf("failed to find the specified datacenter: %w", err)
	}

	if err := c.generateAndCreateSecret(ctx, vsphereSecretPrefixName, c.Credentials.GenerateSecretData(datacenter.Spec.VSphere)); err != nil {
		return fmt.Errorf("failed to create credential secret: %w", err)
	}
	c.log.Debugw("secret created", "name", fmt.Sprintf("%s-%s", vsphereSecretPrefixName, c.name))

	if err := c.generateAndCreateCluster(ctx, kubermaticv1.CloudSpec{
		DatacenterName: c.DatacenterName,
		VSphere: &kubermaticv1.VSphereCloudSpec{
			CredentialsReference: &types.GlobalSecretKeySelector{
				ObjectReference: corev1.ObjectReference{
					Name:      fmt.Sprintf("%s-%s", vsphereSecretPrefixName, c.name),
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

func (c *VsphereClusterJig) CreateMachineDeployment(ctx context.Context, userClient ctrlruntimeclient.Client) error {
	datacenter, err := c.getDatacenter(ctx, c.DatacenterName)
	if err != nil {
		return fmt.Errorf("failed to find the specified datacenter: %w", err)
	}

	if err := c.generateAndCreateMachineDeployment(ctx, userClient, c.Credentials.GenerateProviderSpec(c.name, datacenter.Spec.VSphere)); err != nil {
		return fmt.Errorf("failed to create machine deployment: %w", err)
	}
	return nil
}

func (c *VsphereClusterJig) Cleanup(ctx context.Context, userClient ctrlruntimeclient.Client) error {
	return c.cleanUp(ctx, userClient)
}

func (c *VsphereClusterJig) CheckComponents(ctx context.Context, userClient ctrlruntimeclient.Client) (bool, error) {
	ccmDeploy := &appsv1.Deployment{}
	if err := c.SeedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: fmt.Sprintf("cluster-%s", c.name), Name: vsphereCCMDeploymentName}, ccmDeploy); err != nil {
		return false, fmt.Errorf("failed to get %s deployment: %w", vsphereCCMDeploymentName, err)
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
	Username string
	Password string
}

func (vc *VsphereCredentialsType) GenerateSecretData(datacenter *kubermaticv1.DatacenterSpecVSphere) map[string][]byte {
	return map[string][]byte{
		resources.VsphereUsername:                    []byte(vc.Username),
		resources.VspherePassword:                    []byte(vc.Password),
		resources.VsphereInfraManagementUserUsername: []byte(""),
		resources.VsphereInfraManagementUserPassword: []byte(""),
	}
}

func (vc *VsphereCredentialsType) GenerateProviderSpec(clustername string, datacenter *kubermaticv1.DatacenterSpecVSphere) []byte {
	folder := fmt.Sprintf("%s/%s", datacenter.RootPath, clustername)
	os := types.OperatingSystemUbuntu

	providerSpec, err := json.Marshal(vspheretypes.RawConfig{
		CPUs:           2,
		MemoryMB:       4096,
		DiskSizeGB:     pointer.Int64(10),
		AllowInsecure:  types.ConfigVarBool{Value: pointer.Bool(false)},
		Cluster:        types.ConfigVarString{Value: datacenter.Cluster},
		Datacenter:     types.ConfigVarString{Value: datacenter.Datacenter},
		Datastore:      types.ConfigVarString{Value: datacenter.DefaultDatastore},
		Folder:         types.ConfigVarString{Value: folder},
		TemplateVMName: types.ConfigVarString{Value: datacenter.Templates[os]},
	})
	if err != nil {
		panic(fmt.Sprintf("JSON marshalling failed: %v", err))
	}

	osSpec, err := json.Marshal(ubuntu.Config{})
	if err != nil {
		panic(fmt.Sprintf("JSON marshalling failed: %v", err))
	}

	cfg := types.Config{
		CloudProvider: types.CloudProviderVsphere,
		CloudProviderSpec: runtime.RawExtension{
			Raw: providerSpec,
		},
		OperatingSystem: os,
		OperatingSystemSpec: runtime.RawExtension{
			Raw: osSpec,
		},
	}

	encoded, err := json.Marshal(cfg)
	if err != nil {
		panic(fmt.Sprintf("JSON marshalling failed: %v", err))
	}

	return encoded
}
