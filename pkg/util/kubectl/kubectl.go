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
	"errors"
	"fmt"
	"path/filepath"

	"k8c.io/kubermatic/sdk/v2/semver"
)

const (
	kubectl132 = "kubectl-1.32"
	kubectl133 = "kubectl-1.33"
	kubectl135 = "kubectl-1.35"
)

// BinaryForClusterVersion returns the full path to a kubectl binary
// that shall be used to communicate with a usercluster. An error is
// returned if no suitable kubectl can be determined.
// We take advantage of version skew policy for kubectl, v1.1.1 would
// support v1.2.x and v1.0.x, to ship only mandatory variants for kubectl.
// See https://kubernetes.io/releases/version-skew-policy/#kubectl for
// more information.
func BinaryForClusterVersion(version *semver.Semver) (string, error) {
	var binary string

	switch version.MajorMinor() {
	case "1.31":
		binary = kubectl132
	case "1.32":
		binary = kubectl133
	case "1.33":
		binary = kubectl133
	case "1.34":
		binary = kubectl135
	case "1.35":
		binary = kubectl135
	default:
		return "", fmt.Errorf("unsupported Kubernetes version %v", version)
	}

	return filepath.Join("/usr/local/bin/", binary), nil
}

func VerifyVersionSkew(clusterVersion, kubectlVersion semver.Semver) error {
	clusterMajor := clusterVersion.Semver().Major()
	kubectlMajor := kubectlVersion.Semver().Major()

	if clusterMajor != kubectlMajor {
		return errors.New("major versions are different between cluster and kubectl")
	}

	clusterMinor := clusterVersion.Semver().Minor()
	kubectlMinor := kubectlVersion.Semver().Minor()

	if kubectlMinor < (clusterMinor - 1) {
		return fmt.Errorf("kubectl would support down to v%d.%d, but cluster is %v", kubectlMajor, kubectlMinor-1, clusterVersion.Semver())
	}

	if kubectlMinor > (clusterMinor + 1) {
		return fmt.Errorf("kubectl would support up to v%d.%d, but cluster is %v", kubectlMajor, kubectlMinor+1, clusterVersion.Semver())
	}

	return nil
}
