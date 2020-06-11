// Code generated by go-swagger; DO NOT EDIT.

package settings

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/models"
)

// PatchCurrentUserSettingsReader is a Reader for the PatchCurrentUserSettings structure.
type PatchCurrentUserSettingsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *PatchCurrentUserSettingsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewPatchCurrentUserSettingsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewPatchCurrentUserSettingsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewPatchCurrentUserSettingsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewPatchCurrentUserSettingsOK creates a PatchCurrentUserSettingsOK with default headers values
func NewPatchCurrentUserSettingsOK() *PatchCurrentUserSettingsOK {
	return &PatchCurrentUserSettingsOK{}
}

/*PatchCurrentUserSettingsOK handles this case with default header values.

UserSettings
*/
type PatchCurrentUserSettingsOK struct {
	Payload *models.UserSettings
}

func (o *PatchCurrentUserSettingsOK) Error() string {
	return fmt.Sprintf("[PATCH /api/v1/me/settings][%d] patchCurrentUserSettingsOK  %+v", 200, o.Payload)
}

func (o *PatchCurrentUserSettingsOK) GetPayload() *models.UserSettings {
	return o.Payload
}

func (o *PatchCurrentUserSettingsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.UserSettings)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPatchCurrentUserSettingsUnauthorized creates a PatchCurrentUserSettingsUnauthorized with default headers values
func NewPatchCurrentUserSettingsUnauthorized() *PatchCurrentUserSettingsUnauthorized {
	return &PatchCurrentUserSettingsUnauthorized{}
}

/*PatchCurrentUserSettingsUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type PatchCurrentUserSettingsUnauthorized struct {
}

func (o *PatchCurrentUserSettingsUnauthorized) Error() string {
	return fmt.Sprintf("[PATCH /api/v1/me/settings][%d] patchCurrentUserSettingsUnauthorized ", 401)
}

func (o *PatchCurrentUserSettingsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewPatchCurrentUserSettingsDefault creates a PatchCurrentUserSettingsDefault with default headers values
func NewPatchCurrentUserSettingsDefault(code int) *PatchCurrentUserSettingsDefault {
	return &PatchCurrentUserSettingsDefault{
		_statusCode: code,
	}
}

/*PatchCurrentUserSettingsDefault handles this case with default header values.

errorResponse
*/
type PatchCurrentUserSettingsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the patch current user settings default response
func (o *PatchCurrentUserSettingsDefault) Code() int {
	return o._statusCode
}

func (o *PatchCurrentUserSettingsDefault) Error() string {
	return fmt.Sprintf("[PATCH /api/v1/me/settings][%d] patchCurrentUserSettings default  %+v", o._statusCode, o.Payload)
}

func (o *PatchCurrentUserSettingsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *PatchCurrentUserSettingsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
