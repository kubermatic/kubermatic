package errors

import (
	"fmt"
	"net/http"
)

// HTTPError represents an HTTP server error.
type HTTPError struct {
	code int
	msg  string
}

// Error implements the error interface.
func (err HTTPError) Error() string {
	return err.msg
}

// StatusCode returns the status code for the error
func (err HTTPError) StatusCode() int {
	return err.code
}

// NewNotFound creates a HTTP 404 error for a kind.
func NewNotFound(kind, name string) error {
	return HTTPError{http.StatusNotFound, fmt.Sprintf("%s %q not found", kind, name)}
}

// NewWrongRequest creates a HTTP 400 error, if we got a wrong request type.
func NewWrongRequest(got, want interface{}) error {
	return HTTPError{http.StatusBadRequest, fmt.Sprintf("Got a '%T' request - expected a '%T' request", got, want)}
}

// NewInDcNotFound creates a HTTP 404 error for a kind in a datacenter.
func NewInDcNotFound(kind, dc, name string) error {
	return HTTPError{http.StatusNotFound, fmt.Sprintf("%s %q in dc %q not found", kind, name, dc)}
}

// NewUnknownVersion creates a HTTP 404 error for a kind in a datacenter.
func NewUnknownVersion(version string) error {
	return HTTPError{http.StatusMethodNotAllowed, fmt.Sprintf("Unknown version '%s'", version)}
}

// NewUnknownUpgradePath creates a HTTP 404 error for a kind in a datacenter.
func NewUnknownUpgradePath(from, to string) error {
	return HTTPError{http.StatusNotFound, fmt.Sprintf("There is no upgrade path from version %q to %q", from, to)}
}

// NewBadRequest creates a HTTP 400 error.
func NewBadRequest(msg string, options ...interface{}) error {
	return HTTPError{http.StatusBadRequest, fmt.Sprintf(msg, options...)}
}

// NewConflict creates a HTTP 409 error for a kind in a datacenter.
func NewConflict(kind, dc, name string) error {
	return HTTPError{http.StatusConflict, fmt.Sprintf("%s %q in dc %q already exists", kind, name, dc)}
}

// NewNotAuthorized creates a HTTP 403 error.
func NewNotAuthorized() error {
	return HTTPError{http.StatusForbidden, "not authorized"}
}

// NewNotImplemented creates a HTTP 501 'not implemented' error.
func NewNotImplemented() error {
	return HTTPError{http.StatusNotImplemented, "not implemented"}
}

// NewAlreadyExists creates a HTTP 409 already exists error
func NewAlreadyExists(kind, name string) error {
	return HTTPError{http.StatusConflict, fmt.Sprintf("%s %q already exists", kind, name)}
}
