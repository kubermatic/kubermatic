package handler

import (
	"errors"
	"fmt"

	"github.com/Masterminds/semver"
)

// errVersionSkew denotes an error condition where a given kubelet/controlplane version pair is not supported
type errVersionSkew struct {
	ControlPlane *semver.Version
	Kubelet      *semver.Version
}

// Error returns a string representation of the error
func (e errVersionSkew) Error() string {
	return fmt.Sprintf("kubelet version %s is not compatible with control plane version %s", e.Kubelet, e.ControlPlane)
}

var _ error = errVersionSkew{}

// ensureVersionCompatible checks whether the given kubelet version
// is deemed compatible with the given version of the control plane.
func ensureVersionCompatible(controlPlane *semver.Version, kubelet *semver.Version) error {
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
		return errVersionSkew{
			ControlPlane: controlPlane,
			Kubelet:      kubelet,
		}
	}

	return nil
}
