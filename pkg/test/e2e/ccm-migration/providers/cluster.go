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

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/test/e2e/ccm-migration/utils"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterJigInterface interface {
	Setup(ctx context.Context) error
	CreateMachineDeployment(ctx context.Context, userClient ctrlruntimeclient.Client) error
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

	SeedClient ctrlruntimeclient.Client
}

func (ccj *CommonClusterJig) generateAndCreateCluster(ctx context.Context, cloudSpec kubermaticv1.CloudSpec, projectID string) error {
	cluster := utils.DefaultCluster(ccj.name, ccj.Version, cloudSpec, projectID)

	if err := ccj.SeedClient.Create(ctx, cluster); err != nil {
		return err
	}

	if err := kubermaticv1helper.UpdateClusterStatus(ctx, ccj.SeedClient, cluster, func(c *kubermaticv1.Cluster) {
		c.Status.UserEmail = "e2e@test.com"
	}); err != nil {
		return errors.Wrap(err, "failed to update cluster status")
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
	cluster := &kubermaticv1.Cluster{}
	if err := ccj.SeedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: ccj.name, Namespace: ""}, cluster); err != nil {
		return errors.Wrap(err, "failed to get user cluster")
	}

	// Skip eviction to speed up the clean up process
	nodes := &corev1.NodeList{}
	if err := userClient.List(ctx, nodes); err != nil {
		return errors.Wrap(err, "failed to list user cluster nodes")
	}

	for _, node := range nodes.Items {
		nodeKey := ctrlruntimeclient.ObjectKey{Name: node.Name}

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
			return errors.Wrapf(retErr, "failed to annotate node %s", node.Name)
		}
	}

	// Delete Cluster
	return wait.PollImmediate(utils.UserClusterPollInterval, utils.CustomTestTimeout, func() (bool, error) {
		cluster := &kubermaticv1.Cluster{}
		var err error
		if err = ccj.SeedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: ccj.name, Namespace: ""}, cluster); kerrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, errors.Wrap(err, "failed to retrieve user cluster")
		}
		if cluster.DeletionTimestamp != nil {
			return false, nil
		}
		err = ccj.SeedClient.Delete(ctx, cluster)
		if err != nil {
			return false, errors.Wrap(err, "failed to delete user cluster")
		}
		return false, nil
	})
}

func (ccj *CommonClusterJig) waitForClusterControlPlaneReady(ctx context.Context) error {
	cluster := &kubermaticv1.Cluster{}
	return wait.PollImmediate(utils.ClusterReadinessCheckPeriod, utils.ClusterReadinessTimeout, func() (bool, error) {
		if err := ccj.SeedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: ccj.name}, cluster); err != nil {
			return false, errors.Wrap(err, "failed to get user cluster")
		}
		_, reconciledSuccessfully := kubermaticv1helper.ClusterReconciliationSuccessful(cluster, kubermatic.Versions{}, true)
		return reconciledSuccessfully, nil
	})
}
