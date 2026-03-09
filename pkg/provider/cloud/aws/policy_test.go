//go:build integration

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

package aws

import (
	"bytes"
	"encoding/json"
	"flag"
	"testing"

	testhelper "k8c.io/kubermatic/v2/pkg/test"
)

var (
	update         = flag.Bool("update", false, "update .golden files")
	clusterName    = "kramer"
	dummyAccountID = "000000000000" // Dummy AWS account ID for testing
)

func TestGetControlPlanePolicy(t *testing.T) {
	policy, err := getControlPlanePolicy(clusterName)
	if err != nil {
		t.Fatal(err)
	}

	testGetPolicy(t, clusterName+"-control-plane", policy)
}

func TestGetWorkerPolicy(t *testing.T) {
	policy, err := getWorkerPolicy(clusterName)
	if err != nil {
		t.Fatal(err)
	}

	testGetPolicy(t, clusterName+"-worker", policy)
}

func TestGetAssumeRolePolicy(t *testing.T) {
	t.Run("with-account-id", func(t *testing.T) {
		policy, err := getAssumeRolePolicy(dummyAccountID)
		if err != nil {
			t.Fatal(err)
		}

		testGetPolicy(t, "assume-role-with-account", policy)
	})
}

func testGetPolicy(t *testing.T, identifier string, policy string) {
	v := map[string]interface{}{}
	if err := json.Unmarshal([]byte(policy), &v); err != nil {
		t.Fatalf("the policy does not contain valid json: %v", err)
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(v); err != nil {
		t.Fatalf("Failed to re-encode policy as JSON: %v", err)
	}

	testhelper.CompareOutput(t, identifier, buf.String(), *update, ".json")
}
