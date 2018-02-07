package errors

import "errors"

var (
	// ErrInstanceNotFound tells that the requested instance was not found on the cloud provider
	ErrInstanceNotFound = errors.New("instance not found")
)
