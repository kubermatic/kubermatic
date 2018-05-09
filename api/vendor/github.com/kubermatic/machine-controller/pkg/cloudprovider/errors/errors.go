package errors

import (
	"errors"
	"fmt"

	"github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
)

var (
	// ErrInstanceNotFound tells that the requested instance was not found on the cloud provider
	ErrInstanceNotFound = errors.New("instance not found")
)

// TerminalError is a helper struct that holds errors of type "terminal"
type TerminalError struct {
	Reason  v1alpha1.MachineStatusError
	Message string
}

func (te TerminalError) Error() string {
	return fmt.Sprintf("An error of type = %v, with message = %v occurred", te.Reason, te.Message)
}

// IsTerminalError is a helper function that helps to determine if a given error is terminal
func IsTerminalError(err error) (bool, v1alpha1.MachineStatusError, string) {
	tError, ok := err.(TerminalError)
	if !ok {
		return false, "", ""
	}
	return true, tError.Reason, tError.Message
}
