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
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	awstypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/aws/types"
	"github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	utilcluster "k8c.io/kubermatic/v2/pkg/util/cluster"
	"k8c.io/operating-system-manager/pkg/providerconfig/ubuntu"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	awsSecretPrefixName = "credentials-aws"
)

func NewClusterJigAWS(seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger, version semver.Semver, seedDatacenter string, credentials AWSCredentialsType) *AWSClusterJig {
	return &AWSClusterJig{
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
	_ ClusterJigInterface = &AWSClusterJig{}
)

type AWSClusterJig struct {
	CommonClusterJig

	Credentials AWSCredentialsType
	cluster     kubermaticv1.Cluster
}

func (c *AWSClusterJig) Setup(ctx context.Context) error {
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

	if err := c.generateAndCreateSecret(ctx, awsSecretPrefixName, c.Credentials.GenerateSecretData(datacenter.Spec.AWS)); err != nil {
		return fmt.Errorf("failed to create credential secret: %w", err)
	}
	c.log.Debugw("secret created", "name", fmt.Sprintf("%s-%s", awsSecretPrefixName, c.name))

	if err := c.generateAndCreateCluster(ctx, kubermaticv1.CloudSpec{
		DatacenterName: c.DatacenterName,
		AWS: &kubermaticv1.AWSCloudSpec{
			CredentialsReference: &types.GlobalSecretKeySelector{
				ObjectReference: corev1.ObjectReference{
					Name:      fmt.Sprintf("%s-%s", awsSecretPrefixName, c.name),
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

func (c *AWSClusterJig) CreateMachineDeployment(ctx context.Context, userClient ctrlruntimeclient.Client) error {
	datacenter, err := c.getDatacenter(ctx, c.DatacenterName)
	if err != nil {
		return fmt.Errorf("failed to find the specified datacenter: %w", err)
	}

	if err := c.generateAndCreateMachineDeployment(ctx, userClient, c.Credentials.GenerateProviderSpec(c.cluster.Spec.Cloud.AWS, datacenter.Spec.AWS)); err != nil {
		return fmt.Errorf("failed to create machine deployment: %w", err)
	}
	return nil
}

func (c *AWSClusterJig) Cleanup(ctx context.Context, userClient ctrlruntimeclient.Client) error {
	return c.cleanUp(ctx, userClient)
}

func (c *AWSClusterJig) CheckComponents(ctx context.Context, userClient ctrlruntimeclient.Client) (bool, error) {
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

func (c *AWSClusterJig) Name() string {
	return c.name
}

func (c *AWSClusterJig) Seed() ctrlruntimeclient.Client {
	return c.SeedClient
}

func (c *AWSClusterJig) Log() *zap.SugaredLogger {
	return c.log
}

type AWSCredentialsType struct {
	resources.AWSCredentials
}

func (c *AWSCredentialsType) GenerateSecretData(datacenter *kubermaticv1.DatacenterSpecAWS) map[string][]byte {
	return map[string][]byte{
		"accessKeyId":     []byte(c.AccessKeyID),
		"secretAccessKey": []byte(c.SecretAccessKey),
	}
}

func (c *AWSCredentialsType) GenerateProviderSpec(spec *kubermaticv1.AWSCloudSpec, datacenter *kubermaticv1.DatacenterSpecAWS) []byte {
	os := types.OperatingSystemUbuntu

	providerSpec, err := json.Marshal(awstypes.RawConfig{
		AccessKeyID:          types.ConfigVarString{Value: c.AccessKeyID},
		SecretAccessKey:      types.ConfigVarString{Value: c.SecretAccessKey},
		AssumeRoleARN:        types.ConfigVarString{Value: c.AssumeRoleARN},
		AssumeRoleExternalID: types.ConfigVarString{Value: c.AssumeRoleExternalID},
		VpcID:                types.ConfigVarString{Value: spec.VPCID},
		SecurityGroupIDs:     []types.ConfigVarString{{Value: spec.SecurityGroupID}},
		InstanceProfile:      types.ConfigVarString{Value: spec.InstanceProfileName},
		InstanceType:         types.ConfigVarString{Value: "t2.medium"},
		DiskSize:             100,
		DiskType:             types.ConfigVarString{Value: "gp2"},
		AssignPublicIP:       pointer.Bool(true),
	})
	if err != nil {
		panic(fmt.Sprintf("JSON marshalling failed: %v", err))
	}

	osSpec, err := json.Marshal(ubuntu.Config{})
	if err != nil {
		panic(fmt.Sprintf("JSON marshalling failed: %v", err))
	}

	cfg := types.Config{
		CloudProvider: types.CloudProviderAWS,
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
