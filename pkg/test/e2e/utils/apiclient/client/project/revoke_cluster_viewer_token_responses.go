// Code generated by go-swagger; DO NOT EDIT.

package project

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// RevokeClusterViewerTokenReader is a Reader for the RevokeClusterViewerToken structure.
type RevokeClusterViewerTokenReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *RevokeClusterViewerTokenReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewRevokeClusterViewerTokenOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewRevokeClusterViewerTokenUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewRevokeClusterViewerTokenForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewRevokeClusterViewerTokenDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewRevokeClusterViewerTokenOK creates a RevokeClusterViewerTokenOK with default headers values
func NewRevokeClusterViewerTokenOK() *RevokeClusterViewerTokenOK {
	return &RevokeClusterViewerTokenOK{}
}

/*
RevokeClusterViewerTokenOK describes a response with status code 200, with default header values.

EmptyResponse is a empty response
*/
type RevokeClusterViewerTokenOK struct {
}

// IsSuccess returns true when this revoke cluster viewer token o k response has a 2xx status code
func (o *RevokeClusterViewerTokenOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this revoke cluster viewer token o k response has a 3xx status code
func (o *RevokeClusterViewerTokenOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this revoke cluster viewer token o k response has a 4xx status code
func (o *RevokeClusterViewerTokenOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this revoke cluster viewer token o k response has a 5xx status code
func (o *RevokeClusterViewerTokenOK) IsServerError() bool {
	return false
}

// IsCode returns true when this revoke cluster viewer token o k response a status code equal to that given
func (o *RevokeClusterViewerTokenOK) IsCode(code int) bool {
	return code == 200
}

func (o *RevokeClusterViewerTokenOK) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/viewertoken][%d] revokeClusterViewerTokenOK ", 200)
}

func (o *RevokeClusterViewerTokenOK) String() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/viewertoken][%d] revokeClusterViewerTokenOK ", 200)
}

func (o *RevokeClusterViewerTokenOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewRevokeClusterViewerTokenUnauthorized creates a RevokeClusterViewerTokenUnauthorized with default headers values
func NewRevokeClusterViewerTokenUnauthorized() *RevokeClusterViewerTokenUnauthorized {
	return &RevokeClusterViewerTokenUnauthorized{}
}

/*
RevokeClusterViewerTokenUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type RevokeClusterViewerTokenUnauthorized struct {
}

// IsSuccess returns true when this revoke cluster viewer token unauthorized response has a 2xx status code
func (o *RevokeClusterViewerTokenUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this revoke cluster viewer token unauthorized response has a 3xx status code
func (o *RevokeClusterViewerTokenUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this revoke cluster viewer token unauthorized response has a 4xx status code
func (o *RevokeClusterViewerTokenUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this revoke cluster viewer token unauthorized response has a 5xx status code
func (o *RevokeClusterViewerTokenUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this revoke cluster viewer token unauthorized response a status code equal to that given
func (o *RevokeClusterViewerTokenUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *RevokeClusterViewerTokenUnauthorized) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/viewertoken][%d] revokeClusterViewerTokenUnauthorized ", 401)
}

func (o *RevokeClusterViewerTokenUnauthorized) String() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/viewertoken][%d] revokeClusterViewerTokenUnauthorized ", 401)
}

func (o *RevokeClusterViewerTokenUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewRevokeClusterViewerTokenForbidden creates a RevokeClusterViewerTokenForbidden with default headers values
func NewRevokeClusterViewerTokenForbidden() *RevokeClusterViewerTokenForbidden {
	return &RevokeClusterViewerTokenForbidden{}
}

/*
RevokeClusterViewerTokenForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type RevokeClusterViewerTokenForbidden struct {
}

// IsSuccess returns true when this revoke cluster viewer token forbidden response has a 2xx status code
func (o *RevokeClusterViewerTokenForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this revoke cluster viewer token forbidden response has a 3xx status code
func (o *RevokeClusterViewerTokenForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this revoke cluster viewer token forbidden response has a 4xx status code
func (o *RevokeClusterViewerTokenForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this revoke cluster viewer token forbidden response has a 5xx status code
func (o *RevokeClusterViewerTokenForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this revoke cluster viewer token forbidden response a status code equal to that given
func (o *RevokeClusterViewerTokenForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *RevokeClusterViewerTokenForbidden) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/viewertoken][%d] revokeClusterViewerTokenForbidden ", 403)
}

func (o *RevokeClusterViewerTokenForbidden) String() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/viewertoken][%d] revokeClusterViewerTokenForbidden ", 403)
}

func (o *RevokeClusterViewerTokenForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewRevokeClusterViewerTokenDefault creates a RevokeClusterViewerTokenDefault with default headers values
func NewRevokeClusterViewerTokenDefault(code int) *RevokeClusterViewerTokenDefault {
	return &RevokeClusterViewerTokenDefault{
		_statusCode: code,
	}
}

/*
RevokeClusterViewerTokenDefault describes a response with status code -1, with default header values.

errorResponse
*/
type RevokeClusterViewerTokenDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the revoke cluster viewer token default response
func (o *RevokeClusterViewerTokenDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this revoke cluster viewer token default response has a 2xx status code
func (o *RevokeClusterViewerTokenDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this revoke cluster viewer token default response has a 3xx status code
func (o *RevokeClusterViewerTokenDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this revoke cluster viewer token default response has a 4xx status code
func (o *RevokeClusterViewerTokenDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this revoke cluster viewer token default response has a 5xx status code
func (o *RevokeClusterViewerTokenDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this revoke cluster viewer token default response a status code equal to that given
func (o *RevokeClusterViewerTokenDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *RevokeClusterViewerTokenDefault) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/viewertoken][%d] revokeClusterViewerToken default  %+v", o._statusCode, o.Payload)
}

func (o *RevokeClusterViewerTokenDefault) String() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/viewertoken][%d] revokeClusterViewerToken default  %+v", o._statusCode, o.Payload)
}

func (o *RevokeClusterViewerTokenDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *RevokeClusterViewerTokenDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
