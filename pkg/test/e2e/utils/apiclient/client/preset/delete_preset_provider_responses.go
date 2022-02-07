// Code generated by go-swagger; DO NOT EDIT.

package preset

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// DeletePresetProviderReader is a Reader for the DeletePresetProvider structure.
type DeletePresetProviderReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeletePresetProviderReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewDeletePresetProviderOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewDeletePresetProviderUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewDeletePresetProviderForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 404:
		result := NewDeletePresetProviderNotFound()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewDeletePresetProviderDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewDeletePresetProviderOK creates a DeletePresetProviderOK with default headers values
func NewDeletePresetProviderOK() *DeletePresetProviderOK {
	return &DeletePresetProviderOK{}
}

/* DeletePresetProviderOK describes a response with status code 200, with default header values.

EmptyResponse is a empty response
*/
type DeletePresetProviderOK struct {
}

func (o *DeletePresetProviderOK) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/presets/{preset_name}/provider/{provider_name}][%d] deletePresetProviderOK ", 200)
}

func (o *DeletePresetProviderOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeletePresetProviderUnauthorized creates a DeletePresetProviderUnauthorized with default headers values
func NewDeletePresetProviderUnauthorized() *DeletePresetProviderUnauthorized {
	return &DeletePresetProviderUnauthorized{}
}

/* DeletePresetProviderUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type DeletePresetProviderUnauthorized struct {
}

func (o *DeletePresetProviderUnauthorized) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/presets/{preset_name}/provider/{provider_name}][%d] deletePresetProviderUnauthorized ", 401)
}

func (o *DeletePresetProviderUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeletePresetProviderForbidden creates a DeletePresetProviderForbidden with default headers values
func NewDeletePresetProviderForbidden() *DeletePresetProviderForbidden {
	return &DeletePresetProviderForbidden{}
}

/* DeletePresetProviderForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type DeletePresetProviderForbidden struct {
}

func (o *DeletePresetProviderForbidden) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/presets/{preset_name}/provider/{provider_name}][%d] deletePresetProviderForbidden ", 403)
}

func (o *DeletePresetProviderForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeletePresetProviderNotFound creates a DeletePresetProviderNotFound with default headers values
func NewDeletePresetProviderNotFound() *DeletePresetProviderNotFound {
	return &DeletePresetProviderNotFound{}
}

/* DeletePresetProviderNotFound describes a response with status code 404, with default header values.

EmptyResponse is a empty response
*/
type DeletePresetProviderNotFound struct {
}

func (o *DeletePresetProviderNotFound) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/presets/{preset_name}/provider/{provider_name}][%d] deletePresetProviderNotFound ", 404)
}

func (o *DeletePresetProviderNotFound) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeletePresetProviderDefault creates a DeletePresetProviderDefault with default headers values
func NewDeletePresetProviderDefault(code int) *DeletePresetProviderDefault {
	return &DeletePresetProviderDefault{
		_statusCode: code,
	}
}

/* DeletePresetProviderDefault describes a response with status code -1, with default header values.

errorResponse
*/
type DeletePresetProviderDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the delete preset provider default response
func (o *DeletePresetProviderDefault) Code() int {
	return o._statusCode
}

func (o *DeletePresetProviderDefault) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/presets/{preset_name}/provider/{provider_name}][%d] deletePresetProvider default  %+v", o._statusCode, o.Payload)
}
func (o *DeletePresetProviderDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *DeletePresetProviderDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
