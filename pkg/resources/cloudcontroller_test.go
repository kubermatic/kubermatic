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

package resources

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

func TestIsOTC(t *testing.T) {
	tests := []struct {
		name    string
		authURL string
		want    bool
	}{
		{
			name:    "Nominal",
			authURL: "https://iam.eu-de.otc.t-systems.com/v3",
			want:    true,
		},
		{
			name:    "Same host",
			authURL: "http://iam.eu-de.otc.t-systems.com/v2.0",
			want:    true,
		},
		{
			name:    "Trailing slash",
			authURL: "https://iam.eu-de.otc.t-systems.com/v3/",
			want:    true,
		},
		{
			name:    "Same host",
			authURL: "https://iam.eu-de.otc.t-systems.com:5000/v3",
			want:    false,
		},
		{
			name:    "IP",
			authURL: "http://192.168.2.1:5000/v2.0",
			want:    false,
		},
		{
			name:    "Other provider",
			authURL: "http://identity.provider.org/v3",
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOTC(&kubermaticv1.DatacenterSpecOpenstack{AuthURL: tt.authURL}); got != tt.want {
				t.Errorf("isOTC() = %v, want %v", got, tt.want)
			}
		})
	}
}
