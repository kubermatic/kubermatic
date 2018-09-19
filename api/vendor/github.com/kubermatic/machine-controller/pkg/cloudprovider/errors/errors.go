package errors

import (
	"errors"
	"fmt"

	"sigs.k8s.io/cluster-api/pkg/apis/cluster/common"
)

var (
	// ErrInstanceNotFound tells that the requested instance was not found on the cloud provider
	ErrInstanceNotFound = errors.New("instance not found")
)

// TerminalError is a helper struct that holds errors of type "terminal"
type TerminalError struct {
	Reason  common.MachineStatusError
	Message string
}

func (te TerminalError) Error() string {
	return fmt.Sprintf("An error of type = %v, with message = %v occurred", te.Reason, te.Message)
}

// IsTerminalError is a helper function that helps to determine if a given error is terminal
func IsTerminalError(err error) (bool, common.MachineStatusError, string) {
	tError, ok := err.(TerminalError)
	if !ok {
		return false, "", ""
	}
	return true, tError.Reason, tError.Message
}
