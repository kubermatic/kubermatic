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

package gatekeeper

import (
	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func PodDisruptionBudgetReconciler() reconciling.NamedPodDisruptionBudgetReconcilerFactory {
	return func() (string, reconciling.PodDisruptionBudgetReconciler) {
		return resources.GatekeeperPodDisruptionBudgetName, func(podDisruption *policyv1.PodDisruptionBudget) (*policyv1.PodDisruptionBudget, error) {
			podDisruption.Labels = map[string]string{"gatekeeper.sh/system": "yes"}
			podDisruption.Spec.MinAvailable = &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 1,
			}
			return podDisruption, nil
		}
	}
}
