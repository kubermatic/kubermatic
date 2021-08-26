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
	"os"
	"path/filepath"

	"github.com/Masterminds/semver/v3"
)

// BinaryForClusterVersion returns the full path to a kubectl binary
// that shall be used to communicate with a usercluster. An error is
// returned if no suitable kubectl can be found.
func BinaryForClusterVersion(version *semver.Version) (string, error) {
	var binary string

	switch version.Minor() {
	case 19:
		binary = "kubectl-1.20"
	case 20:
		binary = "kubectl-1.20"
	case 21:
		binary = "kubectl-1.22"
	case 22:
		binary = "kubectl-1.22"
	default:
		return "", fmt.Errorf("unsupported Kubernetes version %v", version)
	}

	fullPath := filepath.Join("/usr/local/bin/", binary)

	if _, err := os.Stat(fullPath); err != nil {
		return "", fmt.Errorf("Kubernetes version %v should use %s, but no such binary was found", version, fullPath)
	}

	return fullPath, nil
}
