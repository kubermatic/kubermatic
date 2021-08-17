// Code generated by go-swagger; DO NOT EDIT.

package admin

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// DeleteAdmissionPluginReader is a Reader for the DeleteAdmissionPlugin structure.
type DeleteAdmissionPluginReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeleteAdmissionPluginReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewDeleteAdmissionPluginOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewDeleteAdmissionPluginUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewDeleteAdmissionPluginForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewDeleteAdmissionPluginDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewDeleteAdmissionPluginOK creates a DeleteAdmissionPluginOK with default headers values
func NewDeleteAdmissionPluginOK() *DeleteAdmissionPluginOK {
	return &DeleteAdmissionPluginOK{}
}

/* DeleteAdmissionPluginOK describes a response with status code 200, with default header values.

EmptyResponse is a empty response
*/
type DeleteAdmissionPluginOK struct {
}

func (o *DeleteAdmissionPluginOK) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/admin/admission/plugins/{name}][%d] deleteAdmissionPluginOK ", 200)
}

func (o *DeleteAdmissionPluginOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteAdmissionPluginUnauthorized creates a DeleteAdmissionPluginUnauthorized with default headers values
func NewDeleteAdmissionPluginUnauthorized() *DeleteAdmissionPluginUnauthorized {
	return &DeleteAdmissionPluginUnauthorized{}
}

/* DeleteAdmissionPluginUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type DeleteAdmissionPluginUnauthorized struct {
}

func (o *DeleteAdmissionPluginUnauthorized) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/admin/admission/plugins/{name}][%d] deleteAdmissionPluginUnauthorized ", 401)
}

func (o *DeleteAdmissionPluginUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteAdmissionPluginForbidden creates a DeleteAdmissionPluginForbidden with default headers values
func NewDeleteAdmissionPluginForbidden() *DeleteAdmissionPluginForbidden {
	return &DeleteAdmissionPluginForbidden{}
}

/* DeleteAdmissionPluginForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type DeleteAdmissionPluginForbidden struct {
}

func (o *DeleteAdmissionPluginForbidden) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/admin/admission/plugins/{name}][%d] deleteAdmissionPluginForbidden ", 403)
}

func (o *DeleteAdmissionPluginForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteAdmissionPluginDefault creates a DeleteAdmissionPluginDefault with default headers values
func NewDeleteAdmissionPluginDefault(code int) *DeleteAdmissionPluginDefault {
	return &DeleteAdmissionPluginDefault{
		_statusCode: code,
	}
}

/* DeleteAdmissionPluginDefault describes a response with status code -1, with default header values.

errorResponse
*/
type DeleteAdmissionPluginDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the delete admission plugin default response
func (o *DeleteAdmissionPluginDefault) Code() int {
	return o._statusCode
}

func (o *DeleteAdmissionPluginDefault) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/admin/admission/plugins/{name}][%d] deleteAdmissionPlugin default  %+v", o._statusCode, o.Payload)
}
func (o *DeleteAdmissionPluginDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *DeleteAdmissionPluginDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
