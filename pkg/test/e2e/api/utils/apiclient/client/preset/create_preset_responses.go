// Code generated by go-swagger; DO NOT EDIT.

package preset

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/api/utils/apiclient/models"
)

// CreatePresetReader is a Reader for the CreatePreset structure.
type CreatePresetReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *CreatePresetReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewCreatePresetOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewCreatePresetUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewCreatePresetForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewCreatePresetDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewCreatePresetOK creates a CreatePresetOK with default headers values
func NewCreatePresetOK() *CreatePresetOK {
	return &CreatePresetOK{}
}

/*CreatePresetOK handles this case with default header values.

Preset
*/
type CreatePresetOK struct {
	Payload *models.Preset
}

func (o *CreatePresetOK) Error() string {
	return fmt.Sprintf("[POST /api/v2/providers/{provider_name}/presets][%d] createPresetOK  %+v", 200, o.Payload)
}

func (o *CreatePresetOK) GetPayload() *models.Preset {
	return o.Payload
}

func (o *CreatePresetOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Preset)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewCreatePresetUnauthorized creates a CreatePresetUnauthorized with default headers values
func NewCreatePresetUnauthorized() *CreatePresetUnauthorized {
	return &CreatePresetUnauthorized{}
}

/*CreatePresetUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type CreatePresetUnauthorized struct {
}

func (o *CreatePresetUnauthorized) Error() string {
	return fmt.Sprintf("[POST /api/v2/providers/{provider_name}/presets][%d] createPresetUnauthorized ", 401)
}

func (o *CreatePresetUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreatePresetForbidden creates a CreatePresetForbidden with default headers values
func NewCreatePresetForbidden() *CreatePresetForbidden {
	return &CreatePresetForbidden{}
}

/*CreatePresetForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type CreatePresetForbidden struct {
}

func (o *CreatePresetForbidden) Error() string {
	return fmt.Sprintf("[POST /api/v2/providers/{provider_name}/presets][%d] createPresetForbidden ", 403)
}

func (o *CreatePresetForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreatePresetDefault creates a CreatePresetDefault with default headers values
func NewCreatePresetDefault(code int) *CreatePresetDefault {
	return &CreatePresetDefault{
		_statusCode: code,
	}
}

/*CreatePresetDefault handles this case with default header values.

errorResponse
*/
type CreatePresetDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the create preset default response
func (o *CreatePresetDefault) Code() int {
	return o._statusCode
}

func (o *CreatePresetDefault) Error() string {
	return fmt.Sprintf("[POST /api/v2/providers/{provider_name}/presets][%d] createPreset default  %+v", o._statusCode, o.Payload)
}

func (o *CreatePresetDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *CreatePresetDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
