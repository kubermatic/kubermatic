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

package util

import (
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SetSeedCondition sets a condition on the given seed using the provided type, status,
// reason and message.
func SetSeedCondition(seed *kubermaticv1.Seed, conditionType kubermaticv1.SeedConditionType, status corev1.ConditionStatus, reason string, message string) {
	newCondition := kubermaticv1.SeedCondition{
		Status:  status,
		Reason:  reason,
		Message: message,
	}

	oldCondition, hadCondition := seed.Status.Conditions[conditionType]
	if hadCondition {
		conditionCopy := oldCondition.DeepCopy()

		// Reset the times before comparing
		conditionCopy.LastHeartbeatTime.Reset()
		conditionCopy.LastTransitionTime.Reset()

		if apiequality.Semantic.DeepEqual(*conditionCopy, newCondition) {
			return
		}
	}

	now := metav1.Now()
	newCondition.LastHeartbeatTime = now
	newCondition.LastTransitionTime = oldCondition.LastTransitionTime
	if hadCondition && oldCondition.Status != status {
		newCondition.LastTransitionTime = now
	}

	if seed.Status.Conditions == nil {
		seed.Status.Conditions = map[kubermaticv1.SeedConditionType]kubermaticv1.SeedCondition{}
	}
	seed.Status.Conditions[conditionType] = newCondition
}
