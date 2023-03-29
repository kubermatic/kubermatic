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

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/test/e2e/jig"

	appsv1 "k8s.io/api/apps/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	osCCMDeploymentName = "openstack-cloud-controller-manager"
)

type OpenStackScenario struct {
	commmonScenario

	credentials jig.OpenStackCredentials
}

var (
	_ TestScenario = &OpenStackScenario{}
)

func NewOpenStackScenario(log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, credentials jig.OpenStackCredentials) *OpenStackScenario {
	return &OpenStackScenario{
		commmonScenario: commmonScenario{
			seedClient: seedClient,
			testJig:    jig.NewOpenStackCluster(seedClient, log, credentials, 1),
		},
		credentials: credentials,
	}
}

func (c *OpenStackScenario) CheckComponents(ctx context.Context, cluster *kubermaticv1.Cluster, userClient ctrlruntimeclient.Client) (bool, error) {
	ccmDeploy := &appsv1.Deployment{}
	if err := c.seedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: fmt.Sprintf("cluster-%s", cluster.Name), Name: osCCMDeploymentName}, ccmDeploy); err != nil {
		return false, fmt.Errorf("failed to get %s deployment: %w", osCCMDeploymentName, err)
	}
	if ccmDeploy.Status.AvailableReplicas == 1 {
		return true, nil
	}

	return false, nil
}
