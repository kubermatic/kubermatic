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

	"github.com/pkg/errors"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/test/e2e/ccm-migration/utils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type CommonClusterJig struct {
	Name           string
	DatacenterName string
	Version        semver.Semver

	SeedClient ctrlruntimeclient.Client
}

func (ccj *CommonClusterJig) generateAndCreateCluster(cloudSpec kubermaticv1.CloudSpec) error {
	cluster := utils.DefaultCluster(ccj.Name, ccj.Version, cloudSpec)
	return ccj.SeedClient.Create(context.TODO(), cluster)
}

func (ccj *CommonClusterJig) generateAndCreateSecret(secretData map[string][]byte) error {
	cluster := utils.DefaultCredentialSecret(ccj.Name, func(secret *corev1.Secret) {
		secret.Data = secretData
	})
	return ccj.SeedClient.Create(context.TODO(), cluster)
}

func (ccj *CommonClusterJig) generateAndCCreateMachineDeployment(userClient ctrlruntimeclient.Client, providerSpec []byte) error {
	machineDeployment := utils.DefaultMachineDeployment(func(md *clusterv1alpha1.MachineDeployment) {
		md.Spec.Template.Spec.ProviderSpec = clusterv1alpha1.ProviderSpec{
			Value: &runtime.RawExtension{
				Raw: providerSpec,
			},
		}
	})
	return userClient.Create(context.TODO(), machineDeployment)
}

func (ccj *CommonClusterJig) waitForClusterControlPlaneReady() error {
	cluster := &kubermaticv1.Cluster{}
	return wait.PollImmediate(utils.ClusterReadinessCheckPeriod, utils.ClusterReadinessTimeout, func() (bool, error) {
		if err := ccj.SeedClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: ccj.Name}, cluster); err != nil {
			return false, errors.Wrap(err, "failed to get user cluster")
		}
		_, cond := kubermaticv1helper.GetClusterCondition(cluster, kubermaticv1.ClusterConditionSeedResourcesUpToDate)
		if cond != nil && cond.Status == corev1.ConditionTrue {
			return true, nil
		}
		return false, nil
	})
}
