// Code generated by go-swagger; DO NOT EDIT.

package allowedregistries

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// DeleteAllowedRegistryReader is a Reader for the DeleteAllowedRegistry structure.
type DeleteAllowedRegistryReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeleteAllowedRegistryReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewDeleteAllowedRegistryOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewDeleteAllowedRegistryUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewDeleteAllowedRegistryForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewDeleteAllowedRegistryDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewDeleteAllowedRegistryOK creates a DeleteAllowedRegistryOK with default headers values
func NewDeleteAllowedRegistryOK() *DeleteAllowedRegistryOK {
	return &DeleteAllowedRegistryOK{}
}

/* DeleteAllowedRegistryOK describes a response with status code 200, with default header values.

EmptyResponse is a empty response
*/
type DeleteAllowedRegistryOK struct {
}

func (o *DeleteAllowedRegistryOK) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/allowedregistries/{allowed_registry}][%d] deleteAllowedRegistryOK ", 200)
}

func (o *DeleteAllowedRegistryOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteAllowedRegistryUnauthorized creates a DeleteAllowedRegistryUnauthorized with default headers values
func NewDeleteAllowedRegistryUnauthorized() *DeleteAllowedRegistryUnauthorized {
	return &DeleteAllowedRegistryUnauthorized{}
}

/* DeleteAllowedRegistryUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type DeleteAllowedRegistryUnauthorized struct {
}

func (o *DeleteAllowedRegistryUnauthorized) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/allowedregistries/{allowed_registry}][%d] deleteAllowedRegistryUnauthorized ", 401)
}

func (o *DeleteAllowedRegistryUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteAllowedRegistryForbidden creates a DeleteAllowedRegistryForbidden with default headers values
func NewDeleteAllowedRegistryForbidden() *DeleteAllowedRegistryForbidden {
	return &DeleteAllowedRegistryForbidden{}
}

/* DeleteAllowedRegistryForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type DeleteAllowedRegistryForbidden struct {
}

func (o *DeleteAllowedRegistryForbidden) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/allowedregistries/{allowed_registry}][%d] deleteAllowedRegistryForbidden ", 403)
}

func (o *DeleteAllowedRegistryForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteAllowedRegistryDefault creates a DeleteAllowedRegistryDefault with default headers values
func NewDeleteAllowedRegistryDefault(code int) *DeleteAllowedRegistryDefault {
	return &DeleteAllowedRegistryDefault{
		_statusCode: code,
	}
}

/* DeleteAllowedRegistryDefault describes a response with status code -1, with default header values.

errorResponse
*/
type DeleteAllowedRegistryDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the delete allowed registry default response
func (o *DeleteAllowedRegistryDefault) Code() int {
	return o._statusCode
}

func (o *DeleteAllowedRegistryDefault) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/allowedregistries/{allowed_registry}][%d] deleteAllowedRegistry default  %+v", o._statusCode, o.Payload)
}
func (o *DeleteAllowedRegistryDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *DeleteAllowedRegistryDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
