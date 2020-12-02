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

package nodeupdate

import (
	"errors"
	"fmt"

	"github.com/Masterminds/semver/v3"
)

// ErrVersionSkew denotes an error condition where a given kubelet/controlplane version pair is not supported
type ErrVersionSkew struct {
	ControlPlane *semver.Version
	Kubelet      *semver.Version
}

// Error returns a string representation of the error
func (e ErrVersionSkew) Error() string {
	return fmt.Sprintf("kubelet version %s is not compatible with control plane version %s", e.Kubelet, e.ControlPlane)
}

var _ error = ErrVersionSkew{}

// EnsureVersionCompatible checks whether the given kubelet version
// is deemed compatible with the given version of the control plane.
func EnsureVersionCompatible(controlPlane *semver.Version, kubelet *semver.Version) error {
	if controlPlane == nil {
		return errors.New("ensureVersionCompatible: controlPlane is nil")
	}

	if kubelet == nil {
		return errors.New("ensureVersionCompatible: kubelet is nil")
	}

	// Kubelet must be the same major version and no more than 2 minor versions behind the control plane.
	// https://kubernetes.io/docs/setup/version-skew-policy/
	// https://github.com/kubernetes/website/blob/076efdf364651859553681a75f60c957de729023/content/en/docs/setup/version-skew-policy.md
	compatible := kubelet.Major() == controlPlane.Major() && kubelet.Minor() >= (controlPlane.Minor()-2) && kubelet.Minor() <= controlPlane.Minor()

	if !compatible {
		return ErrVersionSkew{
			ControlPlane: controlPlane,
			Kubelet:      kubelet,
		}
	}

	return nil
}
