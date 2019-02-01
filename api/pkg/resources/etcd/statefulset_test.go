package etcd

import (
	"flag"
	"fmt"
	"testing"

	testhelper "github.com/kubermatic/kubermatic/api/pkg/test"
)

var update = flag.Bool("update", false, "update .golden files")

func TestGetEtcdCommand(t *testing.T) {

	tests := []struct {
		name                  string
		clusterName           string
		clusterNamespace      string
		migrate               bool
		enableCorruptionCheck bool
	}{
		{
			name:             "no-migration",
			clusterName:      "lg69pmx8wf",
			clusterNamespace: "cluster-lg69pmx8wf",
			migrate:          false,
		},
		{
			name:             "with-migration",
			clusterName:      "62m9k9tqlm",
			clusterNamespace: "cluster-62m9k9tqlm",
			migrate:          true,
		},
		{
			name:                  "with-corruption-flags",
			clusterName:           "lg69pmx8wf",
			clusterNamespace:      "cluster-lg69pmx8wf",
			migrate:               false,
			enableCorruptionCheck: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args, err := getEtcdCommand(test.clusterName, test.clusterNamespace, test.migrate, test.enableCorruptionCheck)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(args) != 3 {
				t.Fatalf("got less arguments than expected. got %d expected %d", len(args), 3)
			}
			cmd := args[2]

			testhelper.CompareOutput(t, fmt.Sprintf("etcd-command-%s", test.name), cmd, *update)
		})
	}
}
