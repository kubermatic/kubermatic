/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package operatingsystem

import (
	"fmt"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/machine-controller/sdk/providerconfig"
)

func TestDefaultSpec(t *testing.T) {
	for _, os := range providerconfig.AllOperatingSystems {
		for _, provider := range kubermaticv1.SupportedProviders {
			t.Run(fmt.Sprintf("%s on %s", os, provider), func(t *testing.T) {
				spec, err := DefaultSpec(os, provider)
				if err != nil {
					t.Fatalf("Failed to create OS spec: %v", err)
				}
				if spec == nil {
					t.Fatal("Did not expect a nil spec, but got one.")
				}
			})
		}
	}

	t.Run("does-not-exist", func(t *testing.T) {
		spec, err := DefaultSpec(providerconfig.OperatingSystem("does-not-exist"), kubermaticv1.AWSCloudProvider)
		if err == nil {
			t.Fatal("Should not have been able to create an OS spec for a bogus OS.")
		}
		if spec != nil {
			t.Fatalf("Spec should have been nil, but got %v", spec)
		}
	})
}
