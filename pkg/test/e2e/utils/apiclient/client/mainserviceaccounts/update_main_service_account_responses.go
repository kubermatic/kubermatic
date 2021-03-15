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

// UpdateMainServiceAccountReader is a Reader for the UpdateMainServiceAccount structure.
type UpdateMainServiceAccountReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *UpdateMainServiceAccountReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewUpdateMainServiceAccountOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewUpdateMainServiceAccountUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewUpdateMainServiceAccountForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewUpdateMainServiceAccountDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewUpdateMainServiceAccountOK creates a UpdateMainServiceAccountOK with default headers values
func NewUpdateMainServiceAccountOK() *UpdateMainServiceAccountOK {
	return &UpdateMainServiceAccountOK{}
}

/*UpdateMainServiceAccountOK handles this case with default header values.

ServiceAccount
*/
type UpdateMainServiceAccountOK struct {
	Payload *models.ServiceAccount
}

func (o *UpdateMainServiceAccountOK) Error() string {
	return fmt.Sprintf("[PUT /api/V2/serviceaccounts/{serviceaccount_id}][%d] updateMainServiceAccountOK  %+v", 200, o.Payload)
}

func (o *UpdateMainServiceAccountOK) GetPayload() *models.ServiceAccount {
	return o.Payload
}

func (o *UpdateMainServiceAccountOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ServiceAccount)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewUpdateMainServiceAccountUnauthorized creates a UpdateMainServiceAccountUnauthorized with default headers values
func NewUpdateMainServiceAccountUnauthorized() *UpdateMainServiceAccountUnauthorized {
	return &UpdateMainServiceAccountUnauthorized{}
}

/*UpdateMainServiceAccountUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type UpdateMainServiceAccountUnauthorized struct {
}

func (o *UpdateMainServiceAccountUnauthorized) Error() string {
	return fmt.Sprintf("[PUT /api/V2/serviceaccounts/{serviceaccount_id}][%d] updateMainServiceAccountUnauthorized ", 401)
}

func (o *UpdateMainServiceAccountUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewUpdateMainServiceAccountForbidden creates a UpdateMainServiceAccountForbidden with default headers values
func NewUpdateMainServiceAccountForbidden() *UpdateMainServiceAccountForbidden {
	return &UpdateMainServiceAccountForbidden{}
}

/*UpdateMainServiceAccountForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type UpdateMainServiceAccountForbidden struct {
}

func (o *UpdateMainServiceAccountForbidden) Error() string {
	return fmt.Sprintf("[PUT /api/V2/serviceaccounts/{serviceaccount_id}][%d] updateMainServiceAccountForbidden ", 403)
}

func (o *UpdateMainServiceAccountForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewUpdateMainServiceAccountDefault creates a UpdateMainServiceAccountDefault with default headers values
func NewUpdateMainServiceAccountDefault(code int) *UpdateMainServiceAccountDefault {
	return &UpdateMainServiceAccountDefault{
		_statusCode: code,
	}
}

/*UpdateMainServiceAccountDefault handles this case with default header values.

errorResponse
*/
type UpdateMainServiceAccountDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the update main service account default response
func (o *UpdateMainServiceAccountDefault) Code() int {
	return o._statusCode
}

func (o *UpdateMainServiceAccountDefault) Error() string {
	return fmt.Sprintf("[PUT /api/V2/serviceaccounts/{serviceaccount_id}][%d] updateMainServiceAccount default  %+v", o._statusCode, o.Payload)
}

func (o *UpdateMainServiceAccountDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *UpdateMainServiceAccountDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
