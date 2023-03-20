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

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	awsCCMDeploymentName = "aws-cloud-controller-manager"
	awsCSIDaemonSetName  = "ebs-csi-node"
)

type AWSScenario struct {
	commmonScenario

	credentials jig.AWSCredentials
}

var (
	_ TestScenario = &AWSScenario{}
)

func NewAWSScenario(log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, credentials jig.AWSCredentials) *AWSScenario {
	return &AWSScenario{
		commmonScenario: commmonScenario{
			seedClient: seedClient,
			testJig:    jig.NewAWSCluster(seedClient, log, credentials, 1, nil),
		},
		credentials: credentials,
	}
}

func (c *AWSScenario) CheckComponents(ctx context.Context, cluster *kubermaticv1.Cluster, userClient ctrlruntimeclient.Client) (bool, error) {
	ccmDeploy := &appsv1.Deployment{}
	if err := c.seedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: cluster.Status.NamespaceName, Name: awsCCMDeploymentName}, ccmDeploy); err != nil {
		return false, fmt.Errorf("failed to get %s deployment: %w", awsCCMDeploymentName, err)
	}
	if ccmDeploy.Status.AvailableReplicas == 1 {
		return true, nil
	}

	nodeDaemonSet := &appsv1.DaemonSet{}
	if err := userClient.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: metav1.NamespaceSystem, Name: awsCSIDaemonSetName}, nodeDaemonSet); err != nil {
		return false, fmt.Errorf("failed to get %s daemonset: %w", awsCSIDaemonSetName, err)
	}

	return nodeDaemonSet.Status.NumberReady == nodeDaemonSet.Status.DesiredNumberScheduled, nil
}
