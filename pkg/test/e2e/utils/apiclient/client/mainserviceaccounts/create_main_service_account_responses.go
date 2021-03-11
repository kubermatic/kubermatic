// Code generated by go-swagger; DO NOT EDIT.

package mainserviceaccounts

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// CreateMainServiceAccountReader is a Reader for the CreateMainServiceAccount structure.
type CreateMainServiceAccountReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *CreateMainServiceAccountReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 201:
		result := NewCreateMainServiceAccountCreated()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewCreateMainServiceAccountUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewCreateMainServiceAccountForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewCreateMainServiceAccountDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewCreateMainServiceAccountCreated creates a CreateMainServiceAccountCreated with default headers values
func NewCreateMainServiceAccountCreated() *CreateMainServiceAccountCreated {
	return &CreateMainServiceAccountCreated{}
}

/*CreateMainServiceAccountCreated handles this case with default header values.

ServiceAccount
*/
type CreateMainServiceAccountCreated struct {
	Payload *models.ServiceAccount
}

func (o *CreateMainServiceAccountCreated) Error() string {
	return fmt.Sprintf("[POST /api/v2/serviceaccounts][%d] createMainServiceAccountCreated  %+v", 201, o.Payload)
}

func (o *CreateMainServiceAccountCreated) GetPayload() *models.ServiceAccount {
	return o.Payload
}

func (o *CreateMainServiceAccountCreated) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ServiceAccount)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewCreateMainServiceAccountUnauthorized creates a CreateMainServiceAccountUnauthorized with default headers values
func NewCreateMainServiceAccountUnauthorized() *CreateMainServiceAccountUnauthorized {
	return &CreateMainServiceAccountUnauthorized{}
}

/*CreateMainServiceAccountUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type CreateMainServiceAccountUnauthorized struct {
}

func (o *CreateMainServiceAccountUnauthorized) Error() string {
	return fmt.Sprintf("[POST /api/v2/serviceaccounts][%d] createMainServiceAccountUnauthorized ", 401)
}

func (o *CreateMainServiceAccountUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateMainServiceAccountForbidden creates a CreateMainServiceAccountForbidden with default headers values
func NewCreateMainServiceAccountForbidden() *CreateMainServiceAccountForbidden {
	return &CreateMainServiceAccountForbidden{}
}

/*CreateMainServiceAccountForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type CreateMainServiceAccountForbidden struct {
}

func (o *CreateMainServiceAccountForbidden) Error() string {
	return fmt.Sprintf("[POST /api/v2/serviceaccounts][%d] createMainServiceAccountForbidden ", 403)
}

func (o *CreateMainServiceAccountForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateMainServiceAccountDefault creates a CreateMainServiceAccountDefault with default headers values
func NewCreateMainServiceAccountDefault(code int) *CreateMainServiceAccountDefault {
	return &CreateMainServiceAccountDefault{
		_statusCode: code,
	}
}

/*CreateMainServiceAccountDefault handles this case with default header values.

errorResponse
*/
type CreateMainServiceAccountDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the create main service account default response
func (o *CreateMainServiceAccountDefault) Code() int {
	return o._statusCode
}

func (o *CreateMainServiceAccountDefault) Error() string {
	return fmt.Sprintf("[POST /api/v2/serviceaccounts][%d] createMainServiceAccount default  %+v", o._statusCode, o.Payload)
}

func (o *CreateMainServiceAccountDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *CreateMainServiceAccountDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
