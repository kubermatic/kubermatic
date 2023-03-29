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

	semverlib "github.com/Masterminds/semver/v3"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
)

func IsSupported(version *semverlib.Version, provider kubermaticv1.CloudProvider, incompatibilities []*ProviderIncompatibility, conditions ...kubermaticv1.ConditionType) (bool, error) {
	return checkProviderCompatibility(version, provider, kubermaticv1.OperationSupport, incompatibilities, conditions...)
}

func checkProviderCompatibility(version *semverlib.Version, provider kubermaticv1.CloudProvider, operation kubermaticv1.OperationType, incompatibilities []*ProviderIncompatibility, conditions ...kubermaticv1.ConditionType) (bool, error) {
	var compatible = true
	var err error
	for _, pi := range incompatibilities {
		// NB: pi.Provider == "" allows applying incompatibilities to all providers.
		if (pi.Provider == provider || pi.Provider == "") && operation == pi.Operation {
			if pi.Condition == kubermaticv1.ConditionAlways {
				compatible, err = CheckUnconstrained(version, pi.Version)
				if err != nil {
					return false, fmt.Errorf("check incompatibility failed")
				}
			} else {
				for _, ic := range conditions {
					if pi.Condition == ic || ic == kubermaticv1.ConditionAlways || pi.Condition == kubermaticv1.ConditionAlways {
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

func CheckUnconstrained(baseVersion *semverlib.Version, version string) (bool, error) {
	c, err := semverlib.NewConstraint(version)
	if err != nil {
		return false, fmt.Errorf("failed to parse to constraint %s: %w", c, err)
	}

	return !c.Check(baseVersion), nil
}
