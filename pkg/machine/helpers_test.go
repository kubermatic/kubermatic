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

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/util/sets"
)

func TestCompleteCloudProviderSpecWithNoInputsAtAll(t *testing.T) {
	excluded := sets.New(
		string(kubermaticv1.CloudProviderFake),
		string(kubermaticv1.CloudProviderBringYourOwn),
	)

	for _, provider := range sets.List(kubermaticv1.AllCloudProviders) {
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
