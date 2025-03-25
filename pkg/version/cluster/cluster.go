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

package cluster

import (
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/cloudcontroller"
)

// GetVersionConditions returns a kubermaticv1.ConditionType list that should be used when checking
// for available versions in a VersionManager instance.
func GetVersionConditions(spec *kubermaticv1.ClusterSpec) []kubermaticv1.ConditionType {
	conditions := []kubermaticv1.ConditionType{}

	// we will not return the external/internal provider condition for providers that do not
	// have a CCM.
	if cloudcontroller.HasCCM(spec) {
		if spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] {
			conditions = append(conditions, kubermaticv1.ExternalCloudProviderCondition)
		} else {
			conditions = append(conditions, kubermaticv1.InTreeCloudProviderCondition)
		}
	}

	return conditions
}
