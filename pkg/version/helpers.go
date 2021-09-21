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

package version

import (
	"fmt"

	"github.com/Masterminds/semver/v3"

	v1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
)

func IsSupported(version *semver.Version, provider kubermaticv1.ProviderType, incompatibilities []*ProviderIncompatibility, conditions ...operatorv1alpha1.ConditionType) (bool, error) {
	return checkProviderCompatibility(version, provider, v1.KubernetesClusterType, operatorv1alpha1.SupportOperation, incompatibilities, conditions...)
}

func checkProviderCompatibility(version *semver.Version, provider kubermaticv1.ProviderType, clusterType string, operation operatorv1alpha1.OperationType, incompatibilities []*ProviderIncompatibility, conditions ...operatorv1alpha1.ConditionType) (bool, error) {
	var compatible = true
	var err error
	for _, pi := range incompatibilities {
		if pi.Provider == provider && pi.Type == clusterType && operation == pi.Operation {
			if pi.Condition == operatorv1alpha1.AlwaysCondition {
				compatible, err = CheckUnconstrained(version, pi.Version)
				if err != nil {
					return false, fmt.Errorf("check incompatibility failed")
				}
			} else {
				for _, ic := range conditions {
					if pi.Condition == ic || ic == operatorv1alpha1.AlwaysCondition || pi.Condition == operatorv1alpha1.AlwaysCondition {
						compatible, err = CheckUnconstrained(version, pi.Version)
						if err != nil {
							return false, fmt.Errorf("check incompatibility failed")
						}
						if !compatible {
							return false, nil
						}
					}
				}
			}
			if !compatible {
				return false, nil
			}
		}
	}
	return compatible, nil
}

func CheckUnconstrained(baseVersion *semver.Version, version string) (bool, error) {
	c, err := semver.NewConstraint(version)
	if err != nil {
		return false, fmt.Errorf("failed to parse to constraint %s: %v", c, err)
	}

	return !c.Check(baseVersion), nil
}
