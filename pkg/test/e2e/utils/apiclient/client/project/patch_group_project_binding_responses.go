// Code generated by go-swagger; DO NOT EDIT.

package project

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// PatchGroupProjectBindingReader is a Reader for the PatchGroupProjectBinding structure.
type PatchGroupProjectBindingReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *PatchGroupProjectBindingReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewPatchGroupProjectBindingOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewPatchGroupProjectBindingUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewPatchGroupProjectBindingForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewPatchGroupProjectBindingDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewPatchGroupProjectBindingOK creates a PatchGroupProjectBindingOK with default headers values
func NewPatchGroupProjectBindingOK() *PatchGroupProjectBindingOK {
	return &PatchGroupProjectBindingOK{}
}

/* PatchGroupProjectBindingOK describes a response with status code 200, with default header values.

EmptyResponse is a empty response
*/
type PatchGroupProjectBindingOK struct {
}

func (o *PatchGroupProjectBindingOK) Error() string {
	return fmt.Sprintf("[PATCH /api/v2/projects/{project_id}/groupbindings/{binding_name}][%d] patchGroupProjectBindingOK ", 200)
}

func (o *PatchGroupProjectBindingOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewPatchGroupProjectBindingUnauthorized creates a PatchGroupProjectBindingUnauthorized with default headers values
func NewPatchGroupProjectBindingUnauthorized() *PatchGroupProjectBindingUnauthorized {
	return &PatchGroupProjectBindingUnauthorized{}
}

/* PatchGroupProjectBindingUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type PatchGroupProjectBindingUnauthorized struct {
}

func (o *PatchGroupProjectBindingUnauthorized) Error() string {
	return fmt.Sprintf("[PATCH /api/v2/projects/{project_id}/groupbindings/{binding_name}][%d] patchGroupProjectBindingUnauthorized ", 401)
}

func (o *PatchGroupProjectBindingUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewPatchGroupProjectBindingForbidden creates a PatchGroupProjectBindingForbidden with default headers values
func NewPatchGroupProjectBindingForbidden() *PatchGroupProjectBindingForbidden {
	return &PatchGroupProjectBindingForbidden{}
}

/* PatchGroupProjectBindingForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type PatchGroupProjectBindingForbidden struct {
}

func (o *PatchGroupProjectBindingForbidden) Error() string {
	return fmt.Sprintf("[PATCH /api/v2/projects/{project_id}/groupbindings/{binding_name}][%d] patchGroupProjectBindingForbidden ", 403)
}

func (o *PatchGroupProjectBindingForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewPatchGroupProjectBindingDefault creates a PatchGroupProjectBindingDefault with default headers values
func NewPatchGroupProjectBindingDefault(code int) *PatchGroupProjectBindingDefault {
	return &PatchGroupProjectBindingDefault{
		_statusCode: code,
	}
}

/* PatchGroupProjectBindingDefault describes a response with status code -1, with default header values.

errorResponse
*/
type PatchGroupProjectBindingDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the patch group project binding default response
func (o *PatchGroupProjectBindingDefault) Code() int {
	return o._statusCode
}

func (o *PatchGroupProjectBindingDefault) Error() string {
	return fmt.Sprintf("[PATCH /api/v2/projects/{project_id}/groupbindings/{binding_name}][%d] patchGroupProjectBinding default  %+v", o._statusCode, o.Payload)
}
func (o *PatchGroupProjectBindingDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *PatchGroupProjectBindingDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

/*PatchGroupProjectBindingBody patch group project binding body
swagger:model PatchGroupProjectBindingBody
*/
type PatchGroupProjectBindingBody struct {

	// role
	Role string `json:"role,omitempty"`
}

// Validate validates this patch group project binding body
func (o *PatchGroupProjectBindingBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this patch group project binding body based on context it is used
func (o *PatchGroupProjectBindingBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *PatchGroupProjectBindingBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *PatchGroupProjectBindingBody) UnmarshalBinary(b []byte) error {
	var res PatchGroupProjectBindingBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}
