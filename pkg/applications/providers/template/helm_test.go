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

package template

import (
	"testing"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetReleaseName(t *testing.T) {
	tests := []struct {
		testName            string
		appNamespace        string
		appName             string
		expectedReleaseName string
	}{
		{
			testName:            "when len (namspaceName) <= 53 then namspaceName is returned",
			appNamespace:        "default",
			appName:             "app1",
			expectedReleaseName: "default-app1",
		},
		{
			testName:            "when len(namspaceName) > 53 and len(appName) <=43  then appName-sha1(namespace)[:9] is returned",
			appNamespace:        "default-012345678901234567890123456789001234567890123456789",
			appName:             "app1",
			expectedReleaseName: "app1-8232574ba",
		},

		{
			testName:            "when len(namspaceName) > 53 and len(appName) >43  then appName[:43]-sha1(namespace)[:9] is returned",
			appNamespace:        "default",
			appName:             "application-installation-super-long-name-that-should-be-trucated",
			expectedReleaseName: "application-installation-super-long-name-th-7505d64a5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			appInstallation := &appskubermaticv1.ApplicationInstallation{ObjectMeta: metav1.ObjectMeta{
				Namespace: tt.appNamespace,
				Name:      tt.appName,
			}}

			res := getReleaseName(appInstallation)
			if res != tt.expectedReleaseName {
				t.Errorf("getReleaseName() = %v, want %v", res, tt.expectedReleaseName)
			}

			size := len(res)
			if size > 53 {
				t.Errorf("getReleaseName() size should be less or equals to 53. got %v", size)
			}

		})
	}
}
