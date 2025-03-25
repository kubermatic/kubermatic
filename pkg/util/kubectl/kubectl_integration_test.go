//go:build integration

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

package kubectl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/defaulting"
)

type kubectlVersionOutput struct {
	ClientVersion struct {
		GitVersion string `json:"gitVersion"`
	} `json:"clientVersion"`
}

func TestVersionSkewIsRespected(t *testing.T) {
	dockerImage := os.Getenv("KUBECTL_TEST_IMAGE")
	if dockerImage == "" {
		t.Skip("No KUBECTL_TEST_IMAGE set, skipping kubectl version skew tests.")
	}

	for _, v := range defaulting.DefaultKubernetesVersioning.Versions {
		t.Run(v.String(), func(t *testing.T) {
			if err := testVersionSkew(v, dockerImage); err != nil {
				t.Errorf("Failed to get a kubectl version that's compatible to cluster version %q: %v", v, err)
			}
		})
	}
}

func testVersionSkew(clusterVersison semver.Semver, dockerImage string) error {
	binary, err := BinaryForClusterVersion(&clusterVersison)
	if err != nil {
		return fmt.Errorf("no kubectl binary found: %w", err)
	}

	cmd := exec.Command("docker",
		"run",
		"--rm",
		dockerImage,
		binary,
		"version",
		"--client",
		"--output",
		"json",
	)

	var buf bytes.Buffer
	cmd.Stdout = &buf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to determine kubectl version: %w", err)
	}

	data := kubectlVersionOutput{}
	if err := json.NewDecoder(&buf).Decode(&data); err != nil {
		return fmt.Errorf("failed to decode kubectl output: %w", err)
	}

	kubectlVersion, err := semver.NewSemver(data.ClientVersion.GitVersion)
	if err != nil {
		return fmt.Errorf("failed to parse %q as a semver: %w", data.ClientVersion.GitVersion, err)
	}

	if err := VerifyVersionSkew(clusterVersison, *kubectlVersion); err != nil {
		return fmt.Errorf("kubectl should have been compatible, but: %w", err)
	}

	return nil
}
