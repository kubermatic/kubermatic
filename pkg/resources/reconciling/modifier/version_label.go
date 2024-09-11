/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package modifier

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// VersionLabel adds the version label for Deployments and their corresponding pods.
func VersionLabel(version string) reconciling.ObjectModifier {
	return func(reconciler reconciling.ObjectReconciler) reconciling.ObjectReconciler {
		return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
			obj, err := reconciler(existing)
			if err != nil {
				return obj, err
			}

			deployment, ok := obj.(*appsv1.Deployment)
			if !ok {
				return obj, fmt.Errorf("VersionLabelModifier is only implemented for deployments, not %T", obj)
			}

			labels := map[string]string{
				resources.VersionLabel: version,
			}

			kubernetes.EnsureLabels(deployment, labels)
			kubernetes.EnsureLabels(&deployment.Spec.Template, labels)

			return obj, nil
		}
	}
}
