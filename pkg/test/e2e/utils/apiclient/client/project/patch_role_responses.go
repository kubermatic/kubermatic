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

// PatchRoleReader is a Reader for the PatchRole structure.
type PatchRoleReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *PatchRoleReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewPatchRoleOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewPatchRoleUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewPatchRoleForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewPatchRoleDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewPatchRoleOK creates a PatchRoleOK with default headers values
func NewPatchRoleOK() *PatchRoleOK {
	return &PatchRoleOK{}
}

/* PatchRoleOK describes a response with status code 200, with default header values.

Role
*/
type PatchRoleOK struct {
	Payload *models.Role
}

// IsSuccess returns true when this patch role o k response has a 2xx status code
func (o *PatchRoleOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this patch role o k response has a 3xx status code
func (o *PatchRoleOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this patch role o k response has a 4xx status code
func (o *PatchRoleOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this patch role o k response has a 5xx status code
func (o *PatchRoleOK) IsServerError() bool {
	return false
}

// IsCode returns true when this patch role o k response a status code equal to that given
func (o *PatchRoleOK) IsCode(code int) bool {
	return code == 200
}

func (o *PatchRoleOK) Error() string {
	return fmt.Sprintf("[PATCH /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/roles/{namespace}/{role_id}][%d] patchRoleOK  %+v", 200, o.Payload)
}

func (o *PatchRoleOK) String() string {
	return fmt.Sprintf("[PATCH /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/roles/{namespace}/{role_id}][%d] patchRoleOK  %+v", 200, o.Payload)
}

func (o *PatchRoleOK) GetPayload() *models.Role {
	return o.Payload
}

func (o *PatchRoleOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Role)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPatchRoleUnauthorized creates a PatchRoleUnauthorized with default headers values
func NewPatchRoleUnauthorized() *PatchRoleUnauthorized {
	return &PatchRoleUnauthorized{}
}

/* PatchRoleUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type PatchRoleUnauthorized struct {
}

// IsSuccess returns true when this patch role unauthorized response has a 2xx status code
func (o *PatchRoleUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this patch role unauthorized response has a 3xx status code
func (o *PatchRoleUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this patch role unauthorized response has a 4xx status code
func (o *PatchRoleUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this patch role unauthorized response has a 5xx status code
func (o *PatchRoleUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this patch role unauthorized response a status code equal to that given
func (o *PatchRoleUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *PatchRoleUnauthorized) Error() string {
	return fmt.Sprintf("[PATCH /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/roles/{namespace}/{role_id}][%d] patchRoleUnauthorized ", 401)
}

func (o *PatchRoleUnauthorized) String() string {
	return fmt.Sprintf("[PATCH /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/roles/{namespace}/{role_id}][%d] patchRoleUnauthorized ", 401)
}

func (o *PatchRoleUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewPatchRoleForbidden creates a PatchRoleForbidden with default headers values
func NewPatchRoleForbidden() *PatchRoleForbidden {
	return &PatchRoleForbidden{}
}

/* PatchRoleForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type PatchRoleForbidden struct {
}

// IsSuccess returns true when this patch role forbidden response has a 2xx status code
func (o *PatchRoleForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this patch role forbidden response has a 3xx status code
func (o *PatchRoleForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this patch role forbidden response has a 4xx status code
func (o *PatchRoleForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this patch role forbidden response has a 5xx status code
func (o *PatchRoleForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this patch role forbidden response a status code equal to that given
func (o *PatchRoleForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *PatchRoleForbidden) Error() string {
	return fmt.Sprintf("[PATCH /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/roles/{namespace}/{role_id}][%d] patchRoleForbidden ", 403)
}

func (o *PatchRoleForbidden) String() string {
	return fmt.Sprintf("[PATCH /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/roles/{namespace}/{role_id}][%d] patchRoleForbidden ", 403)
}

func (o *PatchRoleForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewPatchRoleDefault creates a PatchRoleDefault with default headers values
func NewPatchRoleDefault(code int) *PatchRoleDefault {
	return &PatchRoleDefault{
		_statusCode: code,
	}
}

/* PatchRoleDefault describes a response with status code -1, with default header values.

errorResponse
*/
type PatchRoleDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the patch role default response
func (o *PatchRoleDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this patch role default response has a 2xx status code
func (o *PatchRoleDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this patch role default response has a 3xx status code
func (o *PatchRoleDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this patch role default response has a 4xx status code
func (o *PatchRoleDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this patch role default response has a 5xx status code
func (o *PatchRoleDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this patch role default response a status code equal to that given
func (o *PatchRoleDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *PatchRoleDefault) Error() string {
	return fmt.Sprintf("[PATCH /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/roles/{namespace}/{role_id}][%d] patchRole default  %+v", o._statusCode, o.Payload)
}

func (o *PatchRoleDefault) String() string {
	return fmt.Sprintf("[PATCH /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/roles/{namespace}/{role_id}][%d] patchRole default  %+v", o._statusCode, o.Payload)
}

func (o *PatchRoleDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *PatchRoleDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
