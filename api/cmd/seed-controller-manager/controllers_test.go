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

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"go.uber.org/zap"

	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
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
		log := kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar()
		if err := os.RemoveAll(tempdir); err != nil {
			log.Errorw("Failed to clean up temp dir", zap.Error(err))
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
