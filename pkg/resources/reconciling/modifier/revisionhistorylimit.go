/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// RevisionHistoryLimit returns a modifier that sets the revision history limit for Deployments, StatefulSets and DaemonSets.
func RevisionHistoryLimit(limit int32) reconciling.ObjectModifier {
	return func(reconciler reconciling.ObjectReconciler) reconciling.ObjectReconciler {
		return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
			obj, err := reconciler(existing)
			if err != nil {
				return obj, err
			}

			switch asserted := obj.(type) {
			case *appsv1.Deployment:
				asserted.Spec.RevisionHistoryLimit = ptr.To(limit)
			case *appsv1.StatefulSet:
				asserted.Spec.RevisionHistoryLimit = ptr.To(limit)
			case *appsv1.DaemonSet:
				asserted.Spec.RevisionHistoryLimit = ptr.To(limit)
			default:
				panic(fmt.Sprintf("RevisionHistoryLimit modifier used on incompatible type %T", obj))
			}

			return obj, nil
		}
	}
}
