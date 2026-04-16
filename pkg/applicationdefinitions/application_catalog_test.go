/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package applicationdefinitions

import (
	"io"
	"io/fs"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

func testLogger(t *testing.T) *zap.SugaredLogger {
	return zaptest.NewLogger(t).Sugar()
}

// invalidAppDefFile is an fs.File that returns an ApplicationDefinition with no versions,
// which violates the validation rules.
type invalidAppDefFile struct {
	r io.Reader
}

func (f *invalidAppDefFile) Read(b []byte) (int, error)        { return f.r.Read(b) }
func (f *invalidAppDefFile) Close() error                      { return nil }
func (f *invalidAppDefFile) Stat() (fs.FileInfo, error)        { return nil, nil }

func invalidAppDefFilesFunc() ([]fs.File, error) {
	// A version with no source set triggers "no source provided" validation error.
	yaml := `apiVersion: apps.kubermatic.k8c.io/v1
kind: ApplicationDefinition
metadata:
  name: broken-app
spec:
  method: helm
  versions:
    - version: "v1.0.0"
      template:
        source: {}
`
	return []fs.File{&invalidAppDefFile{r: strings.NewReader(yaml)}}, nil
}

func TestSystemApplicationDefinitionReconcilerFactories_ValidEmbedded(t *testing.T) {
	config := &kubermaticv1.KubermaticConfiguration{}

	_, err := SystemApplicationDefinitionReconcilerFactories(testLogger(t), config, false)
	if err != nil {
		t.Fatalf("expected embedded system ApplicationDefinitions to be valid, got error: %v", err)
	}
}

func TestSystemApplicationDefinitionReconcilerFactories_InvalidDefinition(t *testing.T) {
	original := getSysAppDefFilesFunc
	t.Cleanup(func() { getSysAppDefFilesFunc = original })
	getSysAppDefFilesFunc = invalidAppDefFilesFunc

	config := &kubermaticv1.KubermaticConfiguration{}
	_, err := SystemApplicationDefinitionReconcilerFactories(testLogger(t), config, false)
	if err == nil {
		t.Fatal("expected error for invalid ApplicationDefinition, got nil")
	}
}
