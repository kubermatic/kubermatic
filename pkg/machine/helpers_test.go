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

package machine

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/util/sets"
)

func TestCompleteCloudProviderSpecWithNoInputsAtAll(t *testing.T) {
	excluded := sets.New(
		// external providers
		string(kubermaticv1.AKSCloudProvider),
		string(kubermaticv1.EKSCloudProvider),
		string(kubermaticv1.GKECloudProvider),

		// dummies
		string(kubermaticv1.FakeCloudProvider),
		string(kubermaticv1.BringYourOwnCloudProvider),
		string(kubermaticv1.BaremetalCloudProvider),
		string(kubermaticv1.EdgeCloudProvider),
	)

	for _, provider := range kubermaticv1.SupportedProviders {
		// skip external and fake providers
		if excluded.Has(string(provider)) {
			continue
		}

		t.Run(string(provider), func(t *testing.T) {
			completed, err := CompleteCloudProviderSpec(nil, provider, nil, nil, "")
			if err != nil {
				t.Fatalf("Should not have returned an error, but got: %v", err)
			}

			if completed == nil {
				t.Fatal("Should not have returned nil.")
			}
		})
	}
}
