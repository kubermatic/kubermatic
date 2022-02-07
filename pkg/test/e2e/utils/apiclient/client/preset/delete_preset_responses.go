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

// DeletePresetReader is a Reader for the DeletePreset structure.
type DeletePresetReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeletePresetReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewDeletePresetOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewDeletePresetUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewDeletePresetForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 404:
		result := NewDeletePresetNotFound()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewDeletePresetDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewDeletePresetOK creates a DeletePresetOK with default headers values
func NewDeletePresetOK() *DeletePresetOK {
	return &DeletePresetOK{}
}

/* DeletePresetOK describes a response with status code 200, with default header values.

EmptyResponse is a empty response
*/
type DeletePresetOK struct {
}

func (o *DeletePresetOK) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/presets/{preset_name}][%d] deletePresetOK ", 200)
}

func (o *DeletePresetOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeletePresetUnauthorized creates a DeletePresetUnauthorized with default headers values
func NewDeletePresetUnauthorized() *DeletePresetUnauthorized {
	return &DeletePresetUnauthorized{}
}

/* DeletePresetUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type DeletePresetUnauthorized struct {
}

func (o *DeletePresetUnauthorized) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/presets/{preset_name}][%d] deletePresetUnauthorized ", 401)
}

func (o *DeletePresetUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeletePresetForbidden creates a DeletePresetForbidden with default headers values
func NewDeletePresetForbidden() *DeletePresetForbidden {
	return &DeletePresetForbidden{}
}

/* DeletePresetForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type DeletePresetForbidden struct {
}

func (o *DeletePresetForbidden) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/presets/{preset_name}][%d] deletePresetForbidden ", 403)
}

func (o *DeletePresetForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeletePresetNotFound creates a DeletePresetNotFound with default headers values
func NewDeletePresetNotFound() *DeletePresetNotFound {
	return &DeletePresetNotFound{}
}

/* DeletePresetNotFound describes a response with status code 404, with default header values.

EmptyResponse is a empty response
*/
type DeletePresetNotFound struct {
}

func (o *DeletePresetNotFound) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/presets/{preset_name}][%d] deletePresetNotFound ", 404)
}

func (o *DeletePresetNotFound) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeletePresetDefault creates a DeletePresetDefault with default headers values
func NewDeletePresetDefault(code int) *DeletePresetDefault {
	return &DeletePresetDefault{
		_statusCode: code,
	}
}

/* DeletePresetDefault describes a response with status code -1, with default header values.

errorResponse
*/
type DeletePresetDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the delete preset default response
func (o *DeletePresetDefault) Code() int {
	return o._statusCode
}

func (o *DeletePresetDefault) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/presets/{preset_name}][%d] deletePreset default  %+v", o._statusCode, o.Payload)
}
func (o *DeletePresetDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *DeletePresetDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
