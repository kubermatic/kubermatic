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
