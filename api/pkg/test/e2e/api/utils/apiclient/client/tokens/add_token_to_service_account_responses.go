// Code generated by go-swagger; DO NOT EDIT.

package tokens

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	models "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// AddTokenToServiceAccountReader is a Reader for the AddTokenToServiceAccount structure.
type AddTokenToServiceAccountReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *AddTokenToServiceAccountReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 201:
		result := NewAddTokenToServiceAccountCreated()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	case 401:
		result := NewAddTokenToServiceAccountUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result

	case 403:
		result := NewAddTokenToServiceAccountForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result

	default:
		result := NewAddTokenToServiceAccountDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewAddTokenToServiceAccountCreated creates a AddTokenToServiceAccountCreated with default headers values
func NewAddTokenToServiceAccountCreated() *AddTokenToServiceAccountCreated {
	return &AddTokenToServiceAccountCreated{}
}

/*AddTokenToServiceAccountCreated handles this case with default header values.

ServiceAccountToken
*/
type AddTokenToServiceAccountCreated struct {
	Payload *models.ServiceAccountToken
}

func (o *AddTokenToServiceAccountCreated) Error() string {
	return fmt.Sprintf("[POST /api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}/tokens][%d] addTokenToServiceAccountCreated  %+v", 201, o.Payload)
}

func (o *AddTokenToServiceAccountCreated) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ServiceAccountToken)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewAddTokenToServiceAccountUnauthorized creates a AddTokenToServiceAccountUnauthorized with default headers values
func NewAddTokenToServiceAccountUnauthorized() *AddTokenToServiceAccountUnauthorized {
	return &AddTokenToServiceAccountUnauthorized{}
}

/*AddTokenToServiceAccountUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type AddTokenToServiceAccountUnauthorized struct {
}

func (o *AddTokenToServiceAccountUnauthorized) Error() string {
	return fmt.Sprintf("[POST /api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}/tokens][%d] addTokenToServiceAccountUnauthorized ", 401)
}

func (o *AddTokenToServiceAccountUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewAddTokenToServiceAccountForbidden creates a AddTokenToServiceAccountForbidden with default headers values
func NewAddTokenToServiceAccountForbidden() *AddTokenToServiceAccountForbidden {
	return &AddTokenToServiceAccountForbidden{}
}

/*AddTokenToServiceAccountForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type AddTokenToServiceAccountForbidden struct {
}

func (o *AddTokenToServiceAccountForbidden) Error() string {
	return fmt.Sprintf("[POST /api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}/tokens][%d] addTokenToServiceAccountForbidden ", 403)
}

func (o *AddTokenToServiceAccountForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewAddTokenToServiceAccountDefault creates a AddTokenToServiceAccountDefault with default headers values
func NewAddTokenToServiceAccountDefault(code int) *AddTokenToServiceAccountDefault {
	return &AddTokenToServiceAccountDefault{
		_statusCode: code,
	}
}

/*AddTokenToServiceAccountDefault handles this case with default header values.

ErrorResponse is the default representation of an error
*/
type AddTokenToServiceAccountDefault struct {
	_statusCode int

	Payload *models.ErrorDetails
}

// Code gets the status code for the add token to service account default response
func (o *AddTokenToServiceAccountDefault) Code() int {
	return o._statusCode
}

func (o *AddTokenToServiceAccountDefault) Error() string {
	return fmt.Sprintf("[POST /api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}/tokens][%d] addTokenToServiceAccount default  %+v", o._statusCode, o.Payload)
}

func (o *AddTokenToServiceAccountDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorDetails)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
