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
	"path/filepath"

	"github.com/Masterminds/semver/v3"
)

// BinaryForClusterVersion returns the full path to a kubectl binary
// that shall be used to communicate with a usercluster. An error is
// returned if no suitable kubectl can be determined.
// We take advantage of version skew policy for kubectl, v1.1.1 would support v1.2.x and v1.0.x, to ship
// only mandatory variants for kubectl.
func BinaryForClusterVersion(version *semver.Version) (string, error) {
	var binary string

	switch version.Minor() {
	case 20:
		binary = "kubectl-1.21"
	case 21:
		binary = "kubectl-1.21"
	case 22:
		binary = "kubectl-1.23"
	case 23:
		binary = "kubectl-1.23"
	default:
		return "", fmt.Errorf("unsupported Kubernetes version %v", version)
	}

	return filepath.Join("/usr/local/bin/", binary), nil
}
