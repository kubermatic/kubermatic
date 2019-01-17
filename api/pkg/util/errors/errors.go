package errors

import (
	"fmt"
	"net/http"
)

// HTTPError represents an HTTP server error.
type HTTPError struct {
	code    int
	msg     string
	details string
}

// New creates a brand new HTTPError object
func New(code int, msg string) HTTPError {
	return HTTPError{
		code: code,
		msg:  msg,
	}
}

// NewWithDetails creates a brand new HTTPError object
func NewWithDetails(code int, msg string, details string) HTTPError {
	return HTTPError{
		code:    code,
		msg:     msg,
		details: details,
	}
}

// Error implements the error interface.
func (err HTTPError) Error() string {
	return err.msg
}

// StatusCode returns the status code for the error
func (err HTTPError) StatusCode() int {
	return err.code
}

// Details returns additional message about the error
func (err HTTPError) Details() string {
	return err.details
}

// NewNotFound creates a HTTP 404 error for a kind.
func NewNotFound(kind, name string) error {
	return HTTPError{http.StatusNotFound, fmt.Sprintf("%s %q not found", kind, name), ""}
}

// NewWrongRequest creates a HTTP 400 error, if we got a wrong request type.
func NewWrongRequest(got, want interface{}) error {
	return HTTPError{http.StatusBadRequest, fmt.Sprintf("Got a '%T' request - expected a '%T' request", got, want), ""}
}

// NewBadRequest creates a HTTP 400 error.
func NewBadRequest(msg string, options ...interface{}) error {
	return HTTPError{http.StatusBadRequest, fmt.Sprintf(msg, options...), ""}
}

// NewConflict creates a HTTP 409 error for a kind in a datacenter.
func NewConflict(kind, dc, name string) error {
	return HTTPError{http.StatusConflict, fmt.Sprintf("%s %q in dc %q already exists", kind, name, dc), ""}
}

// NewNotAuthorized creates a HTTP 401 error.
func NewNotAuthorized() error {
	return HTTPError{http.StatusUnauthorized, "not authorized", ""}
}

// NewNotImplemented creates a HTTP 501 'not implemented' error.
func NewNotImplemented() error {
	return HTTPError{http.StatusNotImplemented, "not implemented", ""}
}

// NewAlreadyExists creates a HTTP 409 already exists error
func NewAlreadyExists(kind, name string) error {
	return HTTPError{http.StatusConflict, fmt.Sprintf("%s %q already exists", kind, name), ""}
}
