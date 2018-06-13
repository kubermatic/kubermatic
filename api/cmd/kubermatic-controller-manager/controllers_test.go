package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/golang/glog"
)

func TestGetBackupContainer(t *testing.T) {
	tests := []struct {
		containerYaml string
		errExpected   bool
	}{
		{containerYaml: `
command:
- sleep 5d
image: quay.io/coreos/etcd:v3.2.20
name: etcd`,
			errExpected: false},
		{containerYaml: `
spec: bar`,
			errExpected: true},
	}
	tempdir, err := ioutil.TempDir("/tmp", "kubermatic-cmd-controller-manager-test")
	if err != nil {
		t.Fatalf("Failed to crteate tempdir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempdir); err != nil {
			glog.Errorf("Failed to clean up temp dir: %v", err)
		}
	}()

	for idx, testCase := range tests {
		filepath := fmt.Sprintf("%s/%v.yaml", tempdir, idx)
		if err := ioutil.WriteFile(filepath, []byte(testCase.containerYaml), 0644); err != nil {
			t.Errorf("Failed to write container yaml: %v", err)
		}
		container, err := getContainerFromFile(filepath)
		if err != nil && !testCase.errExpected {
			t.Errorf("Got unexpected error: %v", err)
		}
		if err == nil && testCase.errExpected {
			t.Errorf("Did not get expected error when parsing manifest as container. Manifest:\n%s\nContainer:\n%#v\n",
				testCase.containerYaml, container)
		}
	}

}
