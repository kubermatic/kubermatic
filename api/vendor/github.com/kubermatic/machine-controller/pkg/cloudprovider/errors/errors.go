/*
Copyright 2019 The Machine Controller Authors.

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

package errors

import (
	"errors"
	"fmt"

	"github.com/kubermatic/machine-controller/pkg/apis/cluster/common"
)

var (
	// ErrInstanceNotFound tells that the requested instance was not found on the cloud provider
	ErrInstanceNotFound = errors.New("instance not found")
)

func IsNotFound(err error) bool {
	return err == ErrInstanceNotFound
}

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
