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

package jig

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test"

	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type TestJig struct {
	ProjectJig *ProjectJig
	ClusterJig *ClusterJig
	MachineJig *MachineJig
}

func (j *TestJig) Setup(ctx context.Context, waitMode MachineWaitMode) (*kubermaticv1.Project, *kubermaticv1.Cluster, error) {
	project, err := j.ProjectJig.Create(ctx, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create project: %w", err)
	}

	cluster, err := j.ClusterJig.WithProject(project).Create(ctx, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cluster: %w", err)
	}

	if j.MachineJig != nil {
		err = j.MachineJig.Create(ctx, waitMode, cluster.Spec.Cloud.DatacenterName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create worker nodes: %w", err)
		}
	}

	return project, cluster, nil
}

type ErrorPrinter interface {
	Errorf(format string, args ...interface{})
}

func (j *TestJig) Cleanup(ctx context.Context, t ErrorPrinter, synchronous bool) {
	if j.ClusterJig != nil {
		if err := j.ClusterJig.Delete(ctx, synchronous); err != nil {
			t.Errorf("Failed to delete cluster: %v", err)
		}
	}

	if j.ProjectJig != nil {
		if err := j.ProjectJig.Delete(ctx, synchronous); err != nil {
			t.Errorf("Failed to delete project: %v", err)
		}
	}
}

func (j *TestJig) ClusterClient(ctx context.Context) (ctrlruntimeclient.Client, error) {
	if j.ClusterJig != nil {
		return j.ClusterJig.ClusterClient(ctx)
	}

	return nil, errors.New("no cluster created yet")
}

func (j *TestJig) ClusterRESTConfig(ctx context.Context) (*rest.Config, error) {
	if j.ClusterJig != nil {
		return j.ClusterJig.ClusterRESTConfig(ctx)
	}

	return nil, errors.New("no cluster created yet")
}

func (j *TestJig) WaitForHealthyControlPlane(ctx context.Context, timeout time.Duration) error {
	if j.ClusterJig != nil {
		return j.ClusterJig.WaitForHealthyControlPlane(ctx, timeout)
	}

	return errors.New("no cluster created yet")
}

func NewAlibabaCluster(client ctrlruntimeclient.Client, log *zap.SugaredLogger, credentials AlibabaCredentials, replicas int, spotMaxPriceUSD *string) *TestJig {
	projectJig := NewProjectJig(client, log)

	clusterJig := NewClusterJig(client, log).
		WithHumanReadableName("e2e test cluster").
		WithSSHKeyAgent(false).
		WithCloudSpec(&kubermaticv1.CloudSpec{
			DatacenterName: credentials.KKPDatacenter,
			ProviderName:   string(kubermaticv1.AlibabaCloudProvider),
			Alibaba: &kubermaticv1.AlibabaCloudSpec{
				AccessKeyID:     credentials.AccessKeyID,
				AccessKeySecret: credentials.AccessKeySecret,
			},
		})

	machineJig := NewMachineJig(client, log, nil).
		WithClusterJig(clusterJig).
		WithReplicas(replicas).
		WithAlibaba("ecs.ic5.large", 40)

	return &TestJig{
		ProjectJig: projectJig,
		ClusterJig: clusterJig,
		MachineJig: machineJig,
	}
}

func NewAWSCluster(client ctrlruntimeclient.Client, log *zap.SugaredLogger, credentials AWSCredentials, replicas int, spotMaxPriceUSD *string) *TestJig {
	projectJig := NewProjectJig(client, log)

	clusterJig := NewClusterJig(client, log).
		WithHumanReadableName("e2e test cluster").
		WithSSHKeyAgent(false).
		WithCloudSpec(&kubermaticv1.CloudSpec{
			DatacenterName: credentials.KKPDatacenter,
			ProviderName:   string(kubermaticv1.AWSCloudProvider),
			AWS: &kubermaticv1.AWSCloudSpec{
				AccessKeyID:     credentials.AccessKeyID,
				SecretAccessKey: credentials.SecretAccessKey,
			},
		})

	machineJig := NewMachineJig(client, log, nil).
		WithClusterJig(clusterJig).
		WithReplicas(replicas).
		WithAWS("t3.small", spotMaxPriceUSD)

	return &TestJig{
		ProjectJig: projectJig,
		ClusterJig: clusterJig,
		MachineJig: machineJig,
	}
}

func NewAzureCluster(client ctrlruntimeclient.Client, log *zap.SugaredLogger, credentials AzureCredentials, replicas int) *TestJig {
	projectJig := NewProjectJig(client, log)

	clusterJig := NewClusterJig(client, log).
		WithHumanReadableName("e2e test cluster").
		WithSSHKeyAgent(false).
		WithCloudSpec(&kubermaticv1.CloudSpec{
			DatacenterName: credentials.KKPDatacenter,
			ProviderName:   string(kubermaticv1.AzureCloudProvider),
			Azure: &kubermaticv1.AzureCloudSpec{
				TenantID:       credentials.TenantID,
				SubscriptionID: credentials.SubscriptionID,
				ClientID:       credentials.ClientID,
				ClientSecret:   credentials.ClientSecret,
			},
		})

	machineJig := NewMachineJig(client, log, nil).
		WithClusterJig(clusterJig).
		WithReplicas(replicas).
		WithAzure("Standard_B1ms")

	return &TestJig{
		ProjectJig: projectJig,
		ClusterJig: clusterJig,
		MachineJig: machineJig,
	}
}

func NewHetznerCluster(client ctrlruntimeclient.Client, log *zap.SugaredLogger, credentials HetznerCredentials, replicas int) *TestJig {
	projectJig := NewProjectJig(client, log)

	clusterJig := NewClusterJig(client, log).
		WithHumanReadableName("e2e test cluster").
		WithSSHKeyAgent(false).
		WithCloudSpec(&kubermaticv1.CloudSpec{
			DatacenterName: credentials.KKPDatacenter,
			ProviderName:   string(kubermaticv1.HetznerCloudProvider),
			Hetzner: &kubermaticv1.HetznerCloudSpec{
				Token: credentials.Token,
			},
		})

	machineJig := NewMachineJig(client, log, nil).
		WithClusterJig(clusterJig).
		WithReplicas(replicas).
		WithHetzner("cx21")

	return &TestJig{
		ProjectJig: projectJig,
		ClusterJig: clusterJig,
		MachineJig: machineJig,
	}
}

func NewOpenstackCluster(client ctrlruntimeclient.Client, log *zap.SugaredLogger, credentials OpenstackCredentials, replicas int) *TestJig {
	projectJig := NewProjectJig(client, log)

	clusterJig := NewClusterJig(client, log).
		WithHumanReadableName("e2e test cluster").
		WithSSHKeyAgent(false).
		WithCloudSpec(&kubermaticv1.CloudSpec{
			DatacenterName: credentials.KKPDatacenter,
			ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
			Openstack: &kubermaticv1.OpenstackCloudSpec{
				Username:       credentials.Username,
				Password:       credentials.Password,
				Project:        credentials.Tenant,
				Domain:         credentials.Domain,
				Network:        credentials.Network,
				FloatingIPPool: credentials.FloatingIPPool,
			},
		})

	machineJig := NewMachineJig(client, log, nil).
		WithClusterJig(clusterJig).
		WithReplicas(replicas).
		WithOpenstack("m1.small")

	return &TestJig{
		ProjectJig: projectJig,
		ClusterJig: clusterJig,
		MachineJig: machineJig,
	}
}

func NewVSphereCluster(client ctrlruntimeclient.Client, log *zap.SugaredLogger, credentials VSphereCredentials, replicas int) *TestJig {
	projectJig := NewProjectJig(client, log)

	clusterJig := NewClusterJig(client, log).
		WithHumanReadableName("e2e test cluster").
		WithSSHKeyAgent(false).
		WithCloudSpec(&kubermaticv1.CloudSpec{
			DatacenterName: credentials.KKPDatacenter,
			ProviderName:   string(kubermaticv1.VSphereCloudProvider),
			VSphere: &kubermaticv1.VSphereCloudSpec{
				Username: credentials.Username,
				Password: credentials.Password,
			},
		})

	machineJig := NewMachineJig(client, log, nil).
		WithClusterJig(clusterJig).
		WithReplicas(replicas).
		WithVSphere(2, 4096, 10)

	return &TestJig{
		ProjectJig: projectJig,
		ClusterJig: clusterJig,
		MachineJig: machineJig,
	}
}

func NewDigitaloceanCluster(client ctrlruntimeclient.Client, log *zap.SugaredLogger, credentials DigitaloceanCredentials, replicas int) *TestJig {
	projectJig := NewProjectJig(client, log)

	clusterJig := NewClusterJig(client, log).
		WithHumanReadableName("e2e test cluster").
		WithSSHKeyAgent(false).
		WithCloudSpec(&kubermaticv1.CloudSpec{
			DatacenterName: credentials.KKPDatacenter,
			ProviderName:   string(kubermaticv1.DigitaloceanCloudProvider),
			Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
				Token: credentials.Token,
			},
		})

	machineJig := NewMachineJig(client, log, nil).
		WithClusterJig(clusterJig).
		WithReplicas(replicas).
		WithDigitalocean("c-2")

	return &TestJig{
		ProjectJig: projectJig,
		ClusterJig: clusterJig,
		MachineJig: machineJig,
	}
}

func NewGCPCluster(client ctrlruntimeclient.Client, log *zap.SugaredLogger, credentials GCPCredentials, replicas int) *TestJig {
	projectJig := NewProjectJig(client, log)

	clusterJig := NewClusterJig(client, log).
		WithHumanReadableName("e2e test cluster").
		WithSSHKeyAgent(false).
		WithCloudSpec(&kubermaticv1.CloudSpec{
			DatacenterName: credentials.KKPDatacenter,
			ProviderName:   string(kubermaticv1.GCPCloudProvider),
			GCP: &kubermaticv1.GCPCloudSpec{
				ServiceAccount: test.SafeBase64Encoding(credentials.ServiceAccount),
			},
		})

	machineJig := NewMachineJig(client, log, nil).
		WithClusterJig(clusterJig).
		WithReplicas(replicas).
		WithGCP("e2-small", 25, false)

	return &TestJig{
		ProjectJig: projectJig,
		ClusterJig: clusterJig,
		MachineJig: machineJig,
	}
}

func NewEquinixMetalCluster(client ctrlruntimeclient.Client, log *zap.SugaredLogger, credentials EquinixMetalCredentials, replicas int) *TestJig {
	projectJig := NewProjectJig(client, log)

	clusterJig := NewClusterJig(client, log).
		WithHumanReadableName("e2e test cluster").
		WithSSHKeyAgent(false).
		WithCloudSpec(&kubermaticv1.CloudSpec{
			DatacenterName: credentials.KKPDatacenter,
			ProviderName:   string(kubermaticv1.PacketCloudProvider),
			Packet: &kubermaticv1.PacketCloudSpec{
				APIKey:    credentials.APIKey,
				ProjectID: credentials.ProjectID,
			},
		})

	machineJig := NewMachineJig(client, log, nil).
		WithClusterJig(clusterJig).
		WithReplicas(replicas).
		WithEquinixMetal("c3.small.x86")

	return &TestJig{
		ProjectJig: projectJig,
		ClusterJig: clusterJig,
		MachineJig: machineJig,
	}
}

func NewBYOCluster(client ctrlruntimeclient.Client, log *zap.SugaredLogger, credentials BYOCredentials) *TestJig {
	projectJig := NewProjectJig(client, log)

	clusterJig := NewClusterJig(client, log).
		WithHumanReadableName("e2e test cluster").
		WithSSHKeyAgent(false).
		WithCloudSpec(&kubermaticv1.CloudSpec{
			DatacenterName: credentials.KKPDatacenter,
			ProviderName:   string(kubermaticv1.BringYourOwnCloudProvider),
			BringYourOwn:   &kubermaticv1.BringYourOwnCloudSpec{},
		})

	return &TestJig{
		ProjectJig: projectJig,
		ClusterJig: clusterJig,
	}
}
