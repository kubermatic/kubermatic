// Code generated by go-swagger; DO NOT EDIT.

package serviceaccounts

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// UpdateServiceAccountReader is a Reader for the UpdateServiceAccount structure.
type UpdateServiceAccountReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *UpdateServiceAccountReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewUpdateServiceAccountOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewUpdateServiceAccountUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewUpdateServiceAccountForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewUpdateServiceAccountDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewUpdateServiceAccountOK creates a UpdateServiceAccountOK with default headers values
func NewUpdateServiceAccountOK() *UpdateServiceAccountOK {
	return &UpdateServiceAccountOK{}
}

/* UpdateServiceAccountOK describes a response with status code 200, with default header values.

ServiceAccount
*/
type UpdateServiceAccountOK struct {
	Payload *models.ServiceAccount
}

func (o *UpdateServiceAccountOK) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}][%d] updateServiceAccountOK  %+v", 200, o.Payload)
}
func (o *UpdateServiceAccountOK) GetPayload() *models.ServiceAccount {
	return o.Payload
}

func (o *UpdateServiceAccountOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ServiceAccount)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewUpdateServiceAccountUnauthorized creates a UpdateServiceAccountUnauthorized with default headers values
func NewUpdateServiceAccountUnauthorized() *UpdateServiceAccountUnauthorized {
	return &UpdateServiceAccountUnauthorized{}
}

/* UpdateServiceAccountUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type UpdateServiceAccountUnauthorized struct {
}

func (o *UpdateServiceAccountUnauthorized) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}][%d] updateServiceAccountUnauthorized ", 401)
}

func (o *UpdateServiceAccountUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewUpdateServiceAccountForbidden creates a UpdateServiceAccountForbidden with default headers values
func NewUpdateServiceAccountForbidden() *UpdateServiceAccountForbidden {
	return &UpdateServiceAccountForbidden{}
}

/* UpdateServiceAccountForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type UpdateServiceAccountForbidden struct {
}

func (o *UpdateServiceAccountForbidden) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}][%d] updateServiceAccountForbidden ", 403)
}

func (o *UpdateServiceAccountForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewUpdateServiceAccountDefault creates a UpdateServiceAccountDefault with default headers values
func NewUpdateServiceAccountDefault(code int) *UpdateServiceAccountDefault {
	return &UpdateServiceAccountDefault{
		_statusCode: code,
	}
}

/* UpdateServiceAccountDefault describes a response with status code -1, with default header values.

errorResponse
*/
type UpdateServiceAccountDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the update service account default response
func (o *UpdateServiceAccountDefault) Code() int {
	return o._statusCode
}

func (o *UpdateServiceAccountDefault) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}][%d] updateServiceAccount default  %+v", o._statusCode, o.Payload)
}
func (o *UpdateServiceAccountDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *UpdateServiceAccountDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}