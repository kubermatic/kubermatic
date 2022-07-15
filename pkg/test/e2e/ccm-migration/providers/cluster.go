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
	"time"

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/test/e2e/ccm-migration/utils"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterJigInterface interface {
	Setup(ctx context.Context) error
	CreateMachineDeployment(ctx context.Context, userClient ctrlruntimeclient.Client) error
	WaitForNodeToBeReady(ctx context.Context, userClient ctrlruntimeclient.Client) (bool, error)
	CheckComponents(ctx context.Context, userClient ctrlruntimeclient.Client) (bool, error)
	Cleanup(ctx context.Context, userClient ctrlruntimeclient.Client) error
	Name() string
	Seed() ctrlruntimeclient.Client
	Log() *zap.SugaredLogger
}

type CommonClusterJig struct {
	name           string
	DatacenterName string
	Version        semver.Semver
	log            *zap.SugaredLogger

	SeedClient ctrlruntimeclient.Client
}

func (ccj *CommonClusterJig) getDatacenter(ctx context.Context, datacenter string) (*kubermaticv1.Datacenter, error) {
	seeds := &kubermaticv1.SeedList{}
	if err := ccj.SeedClient.List(ctx, seeds); err != nil {
		return nil, fmt.Errorf("failed to list seeds: %w", err)
	}

	for _, seed := range seeds.Items {
		for name, dc := range seed.Spec.Datacenters {
			if name == datacenter {
				return &dc, nil
			}
		}
	}

	return nil, fmt.Errorf("no Seed contains datacenter %q", datacenter)
}

func (ccj *CommonClusterJig) generateAndCreateCluster(ctx context.Context, cloudSpec kubermaticv1.CloudSpec, projectID string) error {
	cluster := utils.DefaultCluster(ccj.name, ccj.Version, cloudSpec, projectID)

	if err := ccj.SeedClient.Create(ctx, cluster); err != nil {
		return err
	}

	waiter := reconciling.WaitUntilObjectExistsInCacheConditionFunc(ctx, ccj.SeedClient, ccj.log, ctrlruntimeclient.ObjectKeyFromObject(cluster), cluster)
	if err := wait.Poll(100*time.Millisecond, 5*time.Second, waiter); err != nil {
		return fmt.Errorf("failed waiting for the new cluster to appear in the cache: %w", err)
	}

	if err := kubermaticv1helper.UpdateClusterStatus(ctx, ccj.SeedClient, cluster, func(c *kubermaticv1.Cluster) {
		c.Status.UserEmail = "e2e@test.com"
	}); err != nil {
		return fmt.Errorf("failed to update cluster status: %w", err)
	}

	return nil
}

func (ccj *CommonClusterJig) generateAndCreateSecret(ctx context.Context, secretPrefixName string, secretData map[string][]byte) error {
	credentialSecret := utils.DefaultCredentialSecret(fmt.Sprintf("%s-%s", secretPrefixName, ccj.name), func(secret *corev1.Secret) {
		secret.Data = secretData
	})
	return ccj.SeedClient.Create(ctx, credentialSecret)
}

func (ccj *CommonClusterJig) generateAndCreateProject(ctx context.Context, projectName string) error {
	project := utils.DefaultProject(projectName)
	return ccj.SeedClient.Create(ctx, project)
}

func (ccj *CommonClusterJig) generateAndCreateMachineDeployment(ctx context.Context, userClient ctrlruntimeclient.Client, providerSpec []byte) error {
	machineDeployment := utils.DefaultMachineDeployment(func(md *clusterv1alpha1.MachineDeployment) {
		md.Spec.Template.Spec.ProviderSpec = clusterv1alpha1.ProviderSpec{
			Value: &runtime.RawExtension{
				Raw: providerSpec,
			},
		}
		md.Spec.Template.Spec.Versions = clusterv1alpha1.MachineVersionInfo{
			Kubelet: ccj.Version.String(),
		}
	})
	return userClient.Create(ctx, machineDeployment)
}

// CleanUp deletes the cluster.
func (ccj *CommonClusterJig) cleanUp(ctx context.Context, userClient ctrlruntimeclient.Client) error {
	ccj.log.Info("Cleaning up cluster...")

	cluster := &kubermaticv1.Cluster{}
	if err := ccj.SeedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: ccj.name, Namespace: ""}, cluster); err != nil {
		return fmt.Errorf("failed to get user cluster: %w", err)
	}

	// Skip eviction to speed up the clean up process
	nodes := &corev1.NodeList{}
	if err := userClient.List(ctx, nodes); err != nil {
		return fmt.Errorf("failed to list user cluster nodes: %w", err)
	}

	for _, node := range nodes.Items {
		nodeKey := ctrlruntimeclient.ObjectKey{Name: node.Name}
		ccj.log.Debugw("Marking node with skip-eviction...", "node", node.Name)

		retErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			n := corev1.Node{}
			if err := userClient.Get(ctx, nodeKey, &n); err != nil {
				return err
			}

			if n.Annotations == nil {
				n.Annotations = map[string]string{}
			}
			n.Annotations["kubermatic.io/skip-eviction"] = "true"
			return userClient.Update(ctx, &n)
		})
		if retErr != nil {
			return fmt.Errorf("failed to annotate node %s: %w", node.Name, retErr)
		}
	}

	ccj.log.Info("Deleting cluster...")

	// Delete Cluster
	return wait.PollImmediate(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (bool, error) {
		cluster := &kubermaticv1.Cluster{}
		var err error

		// a NotFound error means we're good
		if err = ccj.SeedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: ccj.name}, cluster); apierrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, fmt.Errorf("failed to retrieve user cluster:%w", err)
		}
		if cluster.DeletionTimestamp != nil {
			return false, nil
		}
		err = ccj.SeedClient.Delete(ctx, cluster)
		if err != nil {
			return false, fmt.Errorf("failed to delete user cluster: %w", err)
		}
		return false, nil
	})
}

func (ccj *CommonClusterJig) waitForClusterControlPlaneReady(ctx context.Context) error {
	cluster := &kubermaticv1.Cluster{}
	return wait.PollImmediate(utils.ClusterReadinessCheckPeriod, utils.ClusterReadinessTimeout, func() (bool, error) {
		if err := ccj.SeedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: ccj.name}, cluster); err != nil {
			return false, fmt.Errorf("failed to get user cluster: %w", err)
		}
		_, reconciledSuccessfully := kubermaticv1helper.ClusterReconciliationSuccessful(cluster, kubermatic.Versions{}, true)
		return reconciledSuccessfully, nil
	})
}

func (ccj *CommonClusterJig) WaitForNodeToBeReady(ctx context.Context, userClient ctrlruntimeclient.Client) (bool, error) {
	machines := &clusterv1alpha1.MachineList{}
	if err := userClient.List(ctx, machines); err != nil {
		return false, err
	}
	for _, m := range machines.Items {
		if nodeRef := m.Status.NodeRef; nodeRef != nil && m.DeletionTimestamp == nil {
			node := &corev1.Node{}
			if err := userClient.Get(ctx, types.NamespacedName{Name: nodeRef.Name}, node); err != nil {
				return false, err
			}
			for _, c := range node.Status.Conditions {
				if c.Type == corev1.NodeReady && c.Status == corev1.ConditionTrue {
					return true, nil
				}
			}
		}
	}

	return false, nil
}
