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
	"encoding/json"
	"fmt"

	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// AppliedTolerationsAnnotation records what the Tolerations modifier added. Most reconcilers never
// assign Pod tolerations, so without this record an added toleration could never be removed again.
const AppliedTolerationsAnnotation = "kubermatic.k8c.io/applied-tolerations"

// Tolerations returns a modifier that adds the given tolerations to the Pods of Deployments,
// StatefulSets and DaemonSets. Tolerations set by the reconciler itself are always kept.
func Tolerations(extra []corev1.Toleration) reconciling.ObjectModifier {
	return func(reconciler reconciling.ObjectReconciler) reconciling.ObjectReconciler {
		return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
			obj, err := reconciler(existing)
			if err != nil {
				return obj, err
			}

			var podSpec *corev1.PodSpec

			switch asserted := obj.(type) {
			case *appsv1.Deployment:
				podSpec = &asserted.Spec.Template.Spec
			case *appsv1.StatefulSet:
				podSpec = &asserted.Spec.Template.Spec
			case *appsv1.DaemonSet:
				podSpec = &asserted.Spec.Template.Spec
			default:
				panic(fmt.Sprintf("Tolerations modifier used on incompatible type %T", obj))
			}

			annotations := obj.GetAnnotations()

			tolerations := removeTolerations(podSpec.Tolerations, previouslyAppliedTolerations(annotations))
			tolerations, applied := AppendTolerations(tolerations, extra)
			podSpec.Tolerations = tolerations

			if len(applied) == 0 {
				delete(annotations, AppliedTolerationsAnnotation)
				obj.SetAnnotations(annotations)

				return obj, nil
			}

			encoded, err := json.Marshal(applied)
			if err != nil {
				return obj, fmt.Errorf("failed to encode applied tolerations: %w", err)
			}

			if annotations == nil {
				annotations = map[string]string{}
			}
			annotations[AppliedTolerationsAnnotation] = string(encoded)
			obj.SetAnnotations(annotations)

			return obj, nil
		}
	}
}

// AppendTolerations appends the extra tolerations not already present, returning the combined list
// and those actually added.
func AppendTolerations(existing, extra []corev1.Toleration) (combined, added []corev1.Toleration) {
	combined = existing

	for _, toleration := range extra {
		if containsToleration(combined, toleration) {
			continue
		}

		combined = append(combined, toleration)
		added = append(added, toleration)
	}

	return combined, added
}

func previouslyAppliedTolerations(annotations map[string]string) []corev1.Toleration {
	encoded, ok := annotations[AppliedTolerationsAnnotation]
	if !ok {
		return nil
	}

	// A broken annotation is treated like a missing one and gets overwritten.
	var tolerations []corev1.Toleration
	if err := json.Unmarshal([]byte(encoded), &tolerations); err != nil {
		return nil
	}

	return tolerations
}

func removeTolerations(tolerations, remove []corev1.Toleration) []corev1.Toleration {
	if len(remove) == 0 {
		return tolerations
	}

	var kept []corev1.Toleration
	for _, toleration := range tolerations {
		if !containsToleration(remove, toleration) {
			kept = append(kept, toleration)
		}
	}

	return kept
}

func containsToleration(tolerations []corev1.Toleration, toleration corev1.Toleration) bool {
	for _, candidate := range tolerations {
		if equality.Semantic.DeepEqual(candidate, toleration) {
			return true
		}
	}

	return false
}
