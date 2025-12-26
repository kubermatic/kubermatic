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

package kyverno

import kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

// EnforcementSource indicates where Kyverno enforcement originates from.
type EnforcementSource string

const (
	// EnforcementSourceNone indicates that Kyverno is not enforced.
	EnforcementSourceNone EnforcementSource = ""
	// EnforcementSourceDatacenter indicates that Kyverno is enforced at datacenter level.
	EnforcementSourceDatacenter EnforcementSource = "datacenter"
	// EnforcementSourceSeed indicates that Kyverno is enforced at seed level.
	EnforcementSourceSeed EnforcementSource = "seed"
	// EnforcementSourceGlobal indicates that Kyverno is enforced at global level.
	EnforcementSourceGlobal EnforcementSource = "global"
)

// EnforcementInfo contains resolved enforcement information for Kyverno.
type EnforcementInfo struct {
	// Enforced indicates whether Kyverno enablement is enforced at a parent level.
	Enforced bool

	// Source indicates which level enforces: "datacenter", "seed", "global", or "" if not enforced.
	Source EnforcementSource
}

// GetEnforcement resolves Kyverno enforcement with precedence: DC > Seed > Global.
// Returns enforcement information indicating whether Kyverno is enforced or not.
func GetEnforcement(dcConf, seedConf, globalConf *kubermaticv1.KyvernoConfigurations) EnforcementInfo {
	if dcConf != nil && dcConf.Enforced != nil {
		return EnforcementInfo{
			Enforced: *dcConf.Enforced,
			Source:   EnforcementSourceDatacenter,
		}
	}

	if seedConf != nil && seedConf.Enforced != nil {
		return EnforcementInfo{
			Enforced: *seedConf.Enforced,
			Source:   EnforcementSourceSeed,
		}
	}

	if globalConf != nil && globalConf.Enforced != nil {
		return EnforcementInfo{
			Enforced: *globalConf.Enforced,
			Source:   EnforcementSourceGlobal,
		}
	}

	return EnforcementInfo{
		Enforced: false,
		Source:   EnforcementSourceNone,
	}
}
