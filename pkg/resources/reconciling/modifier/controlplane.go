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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ControlplaneComponent applies a set of common labels to the PodTemplate in a
// replicaing resource (like a Deployment) that lives in a user cluster namespace.
func ControlplaneComponent(cluster *kubermaticv1.Cluster) reconciling.ObjectModifier {
	return func(reconciler reconciling.ObjectReconciler) reconciling.ObjectReconciler {
		return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
			obj, err := reconciler(existing)
			if err != nil {
				return obj, err
			}

			templateLabels := resources.BaseAppLabels(obj.GetName(), nil)
			templateLabels["cluster"] = cluster.Name

			switch asserted := obj.(type) {
			case *appsv1.Deployment:
				kubernetes.EnsureLabels(&asserted.Spec.Template, templateLabels)
			case *appsv1.StatefulSet:
				kubernetes.EnsureLabels(&asserted.Spec.Template, templateLabels)
			case *appsv1.DaemonSet:
				kubernetes.EnsureLabels(&asserted.Spec.Template, templateLabels)
			default:
				panic(fmt.Sprintf("ControlplaneComponent modifier used on incompatible type %T", existing))
			}

			return obj, nil
		}
	}
}
