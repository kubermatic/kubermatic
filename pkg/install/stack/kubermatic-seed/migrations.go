/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package kubermaticseed

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/cluster/client"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/install/stack"

	storagev1 "k8s.io/api/storage/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	cinderCSIDriverName = "cinder.csi.openstack.org"
)

func migrateOpenStackCSIDrivers(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt stack.DeployOptions) error {
	userClusterClientProvider, err := client.NewExternal(kubeClient)
	if err != nil {
		return err
	}

	clusters := kubermaticv1.ClusterList{}
	if err = kubeClient.List(ctx, &clusters); err != nil {
		return err
	}

	for _, cluster := range clusters.Items {
		if cluster.Spec.Cloud.Openstack == nil {
			// at this point we are only looking for openstack clusters
			continue
		}

		userClusterKubeClient, err := userClusterClientProvider.GetClient(ctx, &cluster)
		if err != nil {
			return fmt.Errorf("failed to get %q cluster client: %w", cluster.Name, err)
		}

		// delete named CSIDriver and let addons reconcile it back in current form
		if err = deleteCSIDriver(ctx, userClusterKubeClient, cinderCSIDriverName); err != nil {
			return fmt.Errorf("failed to delete CSIDriver in %q cluster: %w", cluster.Name, err)
		}
	}

	return nil
}

func deleteCSIDriver(ctx context.Context, kubeClient ctrlruntimeclient.Client, name string) error {
	drv := storagev1.CSIDriver{}

	err := kubeClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: name}, &drv)
	if kerrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	return kubeClient.Delete(ctx, &drv)
}

func migrateUserClustersData(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt stack.DeployOptions) error {
	if opt.EnableOpenstackCSIDriverMigration {
		if err := migrateOpenStackCSIDrivers(ctx, logger, kubeClient, opt); err != nil {
			return fmt.Errorf("failed to migrate OpenStack CSIDrivers: %w", err)
		}
	}

	return nil
}
