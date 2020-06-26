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

package aws

import (
	"encoding/json"
	"flag"
	"testing"

	testhelper "github.com/kubermatic/kubermatic/api/pkg/test"
)

var update = flag.Bool("update", false, "update .golden files")

func TestGetPolicy(t *testing.T) {
	clusterName := "cluster-ajcnaw"
	policy, err := getControlPlanePolicy(clusterName)
	if err != nil {
		t.Error(err)
	}

	v := map[string]interface{}{}
	if err := json.Unmarshal([]byte(policy), &v); err != nil {
		t.Errorf("the policy does not contain valid json: %v", err)
	}

	testhelper.CompareOutput(t, clusterName, policy, *update, ".json")
}
