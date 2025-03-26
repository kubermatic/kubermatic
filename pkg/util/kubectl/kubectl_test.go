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

package kubectl

import (
	"fmt"
	"testing"

	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/defaulting"
)

func TestKubectlForAllSupportedVersions(t *testing.T) {
	for _, v := range defaulting.DefaultKubernetesVersioning.Versions {
		_, err := BinaryForClusterVersion(&v)
		if err != nil {
			t.Errorf("No kubectl binary found for cluster version %q: %v", v, err)
		}
	}
}

func TestVerifyVersionSkew(t *testing.T) {
	testcases := []struct {
		cluster string
		kubectl string
		valid   bool
	}{
		// different major versions are never compatible
		{cluster: "1.5", kubectl: "2.5", valid: false},
		{cluster: "2.5", kubectl: "1.5", valid: false},

		// patch releases do not matter
		{cluster: "1.5.100", kubectl: "1.5.999", valid: true},
		{cluster: "1.5.999", kubectl: "1.5.100", valid: true},

		// check actual skew policy
		{cluster: "1.5", kubectl: "1.3", valid: false},
		{cluster: "1.5", kubectl: "1.4", valid: true},
		{cluster: "1.5", kubectl: "1.5", valid: true},
		{cluster: "1.5", kubectl: "1.6", valid: true},
		{cluster: "1.5", kubectl: "1.7", valid: false},
	}

	for _, testcase := range testcases {
		t.Run(fmt.Sprintf("%s vs. %s", testcase.cluster, testcase.kubectl), func(t *testing.T) {
			err := VerifyVersionSkew(*semver.NewSemverOrDie(testcase.cluster), *semver.NewSemverOrDie(testcase.kubectl))

			if testcase.valid && err != nil {
				t.Fatalf("kubectl %s should have been compatible to cluster %s, but got error: %v", testcase.kubectl, testcase.cluster, err)
			}

			if !testcase.valid && err == nil {
				t.Fatalf("kubectl %s should not have been compatible to cluster %s, but got no error", testcase.kubectl, testcase.cluster)
			}
		})
	}
}
