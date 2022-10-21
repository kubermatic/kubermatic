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
	"testing"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

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
		err = j.MachineJig.Create(ctx, waitMode)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create worker nodes: %w", err)
		}
	}

	return project, cluster, nil
}

func (j *TestJig) Cleanup(ctx context.Context, t *testing.T, synchronous bool) {
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

func NewAWSCluster(client ctrlruntimeclient.Client, log *zap.SugaredLogger, accessKeyID, secretAccessKey string, replicas int, spotMaxPriceUSD *string) *TestJig {
	projectJig := NewProjectJig(client, log)

	clusterJig := NewClusterJig(client, log).
		WithHumanReadableName("e2e test cluster").
		WithSSHKeyAgent(false).
		WithCloudSpec(&kubermaticv1.CloudSpec{
			DatacenterName: DatacenterName(),
			ProviderName:   string(kubermaticv1.AWSCloudProvider),
			AWS: &kubermaticv1.AWSCloudSpec{
				SecretAccessKey: secretAccessKey,
				AccessKeyID:     accessKeyID,
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

func NewHetznerCluster(client ctrlruntimeclient.Client, log *zap.SugaredLogger, replicas int) *TestJig {
	projectJig := NewProjectJig(client, log)

	clusterJig := NewClusterJig(client, log).
		WithHumanReadableName("e2e test cluster").
		WithSSHKeyAgent(false).
		WithCloudSpec(&kubermaticv1.CloudSpec{
			DatacenterName: DatacenterName(),
			ProviderName:   string(kubermaticv1.HetznerCloudProvider),
			Hetzner:        &kubermaticv1.HetznerCloudSpec{},
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

func NewBYOCluster(client ctrlruntimeclient.Client, log *zap.SugaredLogger) *TestJig {
	projectJig := NewProjectJig(client, log)

	clusterJig := NewClusterJig(client, log).
		WithHumanReadableName("e2e test cluster").
		WithSSHKeyAgent(false).
		WithCloudSpec(&kubermaticv1.CloudSpec{
			DatacenterName: DatacenterName(),
			ProviderName:   string(kubermaticv1.BringYourOwnCloudProvider),
			BringYourOwn:   &kubermaticv1.BringYourOwnCloudSpec{},
		})

	return &TestJig{
		ProjectJig: projectJig,
		ClusterJig: clusterJig,
	}
}

func NewBYOClusterWithFeatures(client ctrlruntimeclient.Client, log *zap.SugaredLogger, features map[string]bool) *TestJig {
	projectJig := NewProjectJig(client, log)

	clusterJig := NewClusterJig(client, log).
		WithHumanReadableName("e2e test cluster").
		WithSSHKeyAgent(false).
		WithFeatures(features).
		WithCloudSpec(&kubermaticv1.CloudSpec{
			DatacenterName: DatacenterName(),
			ProviderName:   string(kubermaticv1.BringYourOwnCloudProvider),
			BringYourOwn:   &kubermaticv1.BringYourOwnCloudSpec{},
		})

	return &TestJig{
		ProjectJig: projectJig,
		ClusterJig: clusterJig,
	}
}
