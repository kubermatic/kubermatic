// Code generated by go-swagger; DO NOT EDIT.

package project

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	models "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// PatchRoleBindingReader is a Reader for the PatchRoleBinding structure.
type PatchRoleBindingReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *PatchRoleBindingReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewPatchRoleBindingOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	case 401:
		result := NewPatchRoleBindingUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result

	case 403:
		result := NewPatchRoleBindingForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result

	default:
		result := NewPatchRoleBindingDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewPatchRoleBindingOK creates a PatchRoleBindingOK with default headers values
func NewPatchRoleBindingOK() *PatchRoleBindingOK {
	return &PatchRoleBindingOK{}
}

/*PatchRoleBindingOK handles this case with default header values.

RoleBinding
*/
type PatchRoleBindingOK struct {
	Payload *models.RoleBinding
}

func (o *PatchRoleBindingOK) Error() string {
	return fmt.Sprintf("[PATCH /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/roles/{namespace}/{role_id}/bindings/{binding_id}][%d] patchRoleBindingOK  %+v", 200, o.Payload)
}

func (o *PatchRoleBindingOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.RoleBinding)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPatchRoleBindingUnauthorized creates a PatchRoleBindingUnauthorized with default headers values
func NewPatchRoleBindingUnauthorized() *PatchRoleBindingUnauthorized {
	return &PatchRoleBindingUnauthorized{}
}

/*PatchRoleBindingUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type PatchRoleBindingUnauthorized struct {
}

func (o *PatchRoleBindingUnauthorized) Error() string {
	return fmt.Sprintf("[PATCH /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/roles/{namespace}/{role_id}/bindings/{binding_id}][%d] patchRoleBindingUnauthorized ", 401)
}

func (o *PatchRoleBindingUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewPatchRoleBindingForbidden creates a PatchRoleBindingForbidden with default headers values
func NewPatchRoleBindingForbidden() *PatchRoleBindingForbidden {
	return &PatchRoleBindingForbidden{}
}

/*PatchRoleBindingForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type PatchRoleBindingForbidden struct {
}

func (o *PatchRoleBindingForbidden) Error() string {
	return fmt.Sprintf("[PATCH /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/roles/{namespace}/{role_id}/bindings/{binding_id}][%d] patchRoleBindingForbidden ", 403)
}

func (o *PatchRoleBindingForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewPatchRoleBindingDefault creates a PatchRoleBindingDefault with default headers values
func NewPatchRoleBindingDefault(code int) *PatchRoleBindingDefault {
	return &PatchRoleBindingDefault{
		_statusCode: code,
	}
}

/*PatchRoleBindingDefault handles this case with default header values.

errorResponse
*/
type PatchRoleBindingDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the patch role binding default response
func (o *PatchRoleBindingDefault) Code() int {
	return o._statusCode
}

func (o *PatchRoleBindingDefault) Error() string {
	return fmt.Sprintf("[PATCH /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/roles/{namespace}/{role_id}/bindings/{binding_id}][%d] patchRoleBinding default  %+v", o._statusCode, o.Payload)
}

func (o *PatchRoleBindingDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
