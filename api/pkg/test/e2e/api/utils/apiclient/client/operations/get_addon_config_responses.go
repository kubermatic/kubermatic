// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	models "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// GetAddonConfigReader is a Reader for the GetAddonConfig structure.
type GetAddonConfigReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetAddonConfigReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewGetAddonConfigOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	case 401:
		result := NewGetAddonConfigUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result

	default:
		result := NewGetAddonConfigDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetAddonConfigOK creates a GetAddonConfigOK with default headers values
func NewGetAddonConfigOK() *GetAddonConfigOK {
	return &GetAddonConfigOK{}
}

/*GetAddonConfigOK handles this case with default header values.

AddonConfig
*/
type GetAddonConfigOK struct {
	Payload *models.AddonConfig
}

func (o *GetAddonConfigOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/addonconfigs/{addon_id}][%d] getAddonConfigOK  %+v", 200, o.Payload)
}

func (o *GetAddonConfigOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.AddonConfig)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetAddonConfigUnauthorized creates a GetAddonConfigUnauthorized with default headers values
func NewGetAddonConfigUnauthorized() *GetAddonConfigUnauthorized {
	return &GetAddonConfigUnauthorized{}
}

/*GetAddonConfigUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type GetAddonConfigUnauthorized struct {
}

func (o *GetAddonConfigUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/addonconfigs/{addon_id}][%d] getAddonConfigUnauthorized ", 401)
}

func (o *GetAddonConfigUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetAddonConfigDefault creates a GetAddonConfigDefault with default headers values
func NewGetAddonConfigDefault(code int) *GetAddonConfigDefault {
	return &GetAddonConfigDefault{
		_statusCode: code,
	}
}

/*GetAddonConfigDefault handles this case with default header values.

errorResponse
*/
type GetAddonConfigDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the get addon config default response
func (o *GetAddonConfigDefault) Code() int {
	return o._statusCode
}

func (o *GetAddonConfigDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/addonconfigs/{addon_id}][%d] getAddonConfig default  %+v", o._statusCode, o.Payload)
}

func (o *GetAddonConfigDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
