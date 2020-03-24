// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// ListAddonConfigsReader is a Reader for the ListAddonConfigs structure.
type ListAddonConfigsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListAddonConfigsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListAddonConfigsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListAddonConfigsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListAddonConfigsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListAddonConfigsOK creates a ListAddonConfigsOK with default headers values
func NewListAddonConfigsOK() *ListAddonConfigsOK {
	return &ListAddonConfigsOK{}
}

/*ListAddonConfigsOK handles this case with default header values.

AddonConfig
*/
type ListAddonConfigsOK struct {
	Payload []*models.AddonConfig
}

func (o *ListAddonConfigsOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/addonconfigs][%d] listAddonConfigsOK  %+v", 200, o.Payload)
}

func (o *ListAddonConfigsOK) GetPayload() []*models.AddonConfig {
	return o.Payload
}

func (o *ListAddonConfigsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListAddonConfigsUnauthorized creates a ListAddonConfigsUnauthorized with default headers values
func NewListAddonConfigsUnauthorized() *ListAddonConfigsUnauthorized {
	return &ListAddonConfigsUnauthorized{}
}

/*ListAddonConfigsUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type ListAddonConfigsUnauthorized struct {
}

func (o *ListAddonConfigsUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/addonconfigs][%d] listAddonConfigsUnauthorized ", 401)
}

func (o *ListAddonConfigsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListAddonConfigsDefault creates a ListAddonConfigsDefault with default headers values
func NewListAddonConfigsDefault(code int) *ListAddonConfigsDefault {
	return &ListAddonConfigsDefault{
		_statusCode: code,
	}
}

/*ListAddonConfigsDefault handles this case with default header values.

errorResponse
*/
type ListAddonConfigsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list addon configs default response
func (o *ListAddonConfigsDefault) Code() int {
	return o._statusCode
}

func (o *ListAddonConfigsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/addonconfigs][%d] listAddonConfigs default  %+v", o._statusCode, o.Payload)
}

func (o *ListAddonConfigsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListAddonConfigsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
