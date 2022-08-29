// Code generated by go-swagger; DO NOT EDIT.

package admin

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// SetAdminReader is a Reader for the SetAdmin structure.
type SetAdminReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *SetAdminReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewSetAdminOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewSetAdminUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewSetAdminForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewSetAdminDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewSetAdminOK creates a SetAdminOK with default headers values
func NewSetAdminOK() *SetAdminOK {
	return &SetAdminOK{}
}

/*
SetAdminOK describes a response with status code 200, with default header values.

Admin
*/
type SetAdminOK struct {
	Payload *models.Admin
}

// IsSuccess returns true when this set admin o k response has a 2xx status code
func (o *SetAdminOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this set admin o k response has a 3xx status code
func (o *SetAdminOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this set admin o k response has a 4xx status code
func (o *SetAdminOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this set admin o k response has a 5xx status code
func (o *SetAdminOK) IsServerError() bool {
	return false
}

// IsCode returns true when this set admin o k response a status code equal to that given
func (o *SetAdminOK) IsCode(code int) bool {
	return code == 200
}

func (o *SetAdminOK) Error() string {
	return fmt.Sprintf("[PUT /api/v1/admin][%d] setAdminOK  %+v", 200, o.Payload)
}

func (o *SetAdminOK) String() string {
	return fmt.Sprintf("[PUT /api/v1/admin][%d] setAdminOK  %+v", 200, o.Payload)
}

func (o *SetAdminOK) GetPayload() *models.Admin {
	return o.Payload
}

func (o *SetAdminOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Admin)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewSetAdminUnauthorized creates a SetAdminUnauthorized with default headers values
func NewSetAdminUnauthorized() *SetAdminUnauthorized {
	return &SetAdminUnauthorized{}
}

/*
SetAdminUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type SetAdminUnauthorized struct {
}

// IsSuccess returns true when this set admin unauthorized response has a 2xx status code
func (o *SetAdminUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this set admin unauthorized response has a 3xx status code
func (o *SetAdminUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this set admin unauthorized response has a 4xx status code
func (o *SetAdminUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this set admin unauthorized response has a 5xx status code
func (o *SetAdminUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this set admin unauthorized response a status code equal to that given
func (o *SetAdminUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *SetAdminUnauthorized) Error() string {
	return fmt.Sprintf("[PUT /api/v1/admin][%d] setAdminUnauthorized ", 401)
}

func (o *SetAdminUnauthorized) String() string {
	return fmt.Sprintf("[PUT /api/v1/admin][%d] setAdminUnauthorized ", 401)
}

func (o *SetAdminUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewSetAdminForbidden creates a SetAdminForbidden with default headers values
func NewSetAdminForbidden() *SetAdminForbidden {
	return &SetAdminForbidden{}
}

/*
SetAdminForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type SetAdminForbidden struct {
}

// IsSuccess returns true when this set admin forbidden response has a 2xx status code
func (o *SetAdminForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this set admin forbidden response has a 3xx status code
func (o *SetAdminForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this set admin forbidden response has a 4xx status code
func (o *SetAdminForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this set admin forbidden response has a 5xx status code
func (o *SetAdminForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this set admin forbidden response a status code equal to that given
func (o *SetAdminForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *SetAdminForbidden) Error() string {
	return fmt.Sprintf("[PUT /api/v1/admin][%d] setAdminForbidden ", 403)
}

func (o *SetAdminForbidden) String() string {
	return fmt.Sprintf("[PUT /api/v1/admin][%d] setAdminForbidden ", 403)
}

func (o *SetAdminForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewSetAdminDefault creates a SetAdminDefault with default headers values
func NewSetAdminDefault(code int) *SetAdminDefault {
	return &SetAdminDefault{
		_statusCode: code,
	}
}

/*
SetAdminDefault describes a response with status code -1, with default header values.

errorResponse
*/
type SetAdminDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the set admin default response
func (o *SetAdminDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this set admin default response has a 2xx status code
func (o *SetAdminDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this set admin default response has a 3xx status code
func (o *SetAdminDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this set admin default response has a 4xx status code
func (o *SetAdminDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this set admin default response has a 5xx status code
func (o *SetAdminDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this set admin default response a status code equal to that given
func (o *SetAdminDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *SetAdminDefault) Error() string {
	return fmt.Sprintf("[PUT /api/v1/admin][%d] setAdmin default  %+v", o._statusCode, o.Payload)
}

func (o *SetAdminDefault) String() string {
	return fmt.Sprintf("[PUT /api/v1/admin][%d] setAdmin default  %+v", o._statusCode, o.Payload)
}

func (o *SetAdminDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *SetAdminDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
