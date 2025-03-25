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

package operatingsystemprofilesynchronizer

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	ControllerName = "kkp-operating-system-profile-synchronizer"

	operatingSystemManagerAPIVersion = "operatingsystemmanager.k8c.io/v1alpha1"
	customOperatingSystemProfileKind = "CustomOperatingSystemProfile"
	operatingSystemProfileKind       = "OperatingSystemProfile"
)

// UserClusterClientProvider provides functionality to get a user cluster client.
type UserClusterClientProvider interface {
	GetClient(ctx context.Context, c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

func controllerName(subname string) string {
	return fmt.Sprintf("kkp-operating-system-profile-%s", subname)
}

func Add(
	mgr manager.Manager,
	userClusterConnectionProvider UserClusterClientProvider,
	log *zap.SugaredLogger,
	workerName string,
	namespace string,
	numWorkers int,
) error {
	if err := addClusterInitReconciler(mgr, userClusterConnectionProvider, log, workerName, namespace, numWorkers); err != nil {
		return fmt.Errorf("failed to setup cluster init reconciler: %w", err)
	}

	if err := addSyncReconciler(mgr, userClusterConnectionProvider, log, workerName, namespace, numWorkers); err != nil {
		return fmt.Errorf("failed to setup sync reconciler: %w", err)
	}

	return nil
}

func ospReconciler(osp *osmv1alpha1.OperatingSystemProfile) reconciling.NamedOperatingSystemProfileReconcilerFactory {
	return func() (string, reconciling.OperatingSystemProfileReconciler) {
		return osp.Name, func(existing *osmv1alpha1.OperatingSystemProfile) (*osmv1alpha1.OperatingSystemProfile, error) {
			// We need to check if the existing OperatingSystemProfile can be updated.
			// OSP is immutable by nature and to make modifications a version bump is mandatory,
			// so we only update the OSP if the version is different.
			if existing.Spec.Version != osp.Spec.Version {
				existing.Spec = osp.Spec
			}

			return existing, nil
		}
	}
}

func customOSPToOSP(u *unstructured.Unstructured) (*osmv1alpha1.OperatingSystemProfile, error) {
	osp := &osmv1alpha1.OperatingSystemProfile{}
	// Required for converting CustomOperatingSystemProfile to OperatingSystemProfile.
	obj := u.DeepCopy()
	obj.SetKind(operatingSystemProfileKind)
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, osp); err != nil {
		return osp, fmt.Errorf("failed to decode CustomOperatingSystemProfile: %w", err)
	}
	return osp, nil
}
