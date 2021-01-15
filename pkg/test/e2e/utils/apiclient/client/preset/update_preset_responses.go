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

// UpdatePresetReader is a Reader for the UpdatePreset structure.
type UpdatePresetReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *UpdatePresetReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewUpdatePresetOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewUpdatePresetUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewUpdatePresetForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewUpdatePresetDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewUpdatePresetOK creates a UpdatePresetOK with default headers values
func NewUpdatePresetOK() *UpdatePresetOK {
	return &UpdatePresetOK{}
}

/*UpdatePresetOK handles this case with default header values.

Preset
*/
type UpdatePresetOK struct {
	Payload *models.Preset
}

func (o *UpdatePresetOK) Error() string {
	return fmt.Sprintf("[PUT /api/v2/providers/{provider_name}/presets][%d] updatePresetOK  %+v", 200, o.Payload)
}

func (o *UpdatePresetOK) GetPayload() *models.Preset {
	return o.Payload
}

func (o *UpdatePresetOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Preset)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewUpdatePresetUnauthorized creates a UpdatePresetUnauthorized with default headers values
func NewUpdatePresetUnauthorized() *UpdatePresetUnauthorized {
	return &UpdatePresetUnauthorized{}
}

/*UpdatePresetUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type UpdatePresetUnauthorized struct {
}

func (o *UpdatePresetUnauthorized) Error() string {
	return fmt.Sprintf("[PUT /api/v2/providers/{provider_name}/presets][%d] updatePresetUnauthorized ", 401)
}

func (o *UpdatePresetUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewUpdatePresetForbidden creates a UpdatePresetForbidden with default headers values
func NewUpdatePresetForbidden() *UpdatePresetForbidden {
	return &UpdatePresetForbidden{}
}

/*UpdatePresetForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type UpdatePresetForbidden struct {
}

func (o *UpdatePresetForbidden) Error() string {
	return fmt.Sprintf("[PUT /api/v2/providers/{provider_name}/presets][%d] updatePresetForbidden ", 403)
}

func (o *UpdatePresetForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewUpdatePresetDefault creates a UpdatePresetDefault with default headers values
func NewUpdatePresetDefault(code int) *UpdatePresetDefault {
	return &UpdatePresetDefault{
		_statusCode: code,
	}
}

/*UpdatePresetDefault handles this case with default header values.

errorResponse
*/
type UpdatePresetDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the update preset default response
func (o *UpdatePresetDefault) Code() int {
	return o._statusCode
}

func (o *UpdatePresetDefault) Error() string {
	return fmt.Sprintf("[PUT /api/v2/providers/{provider_name}/presets][%d] updatePreset default  %+v", o._statusCode, o.Payload)
}

func (o *UpdatePresetDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *UpdatePresetDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
