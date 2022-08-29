// Code generated by go-swagger; DO NOT EDIT.

package ipampool

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// CreateIPAMPoolReader is a Reader for the CreateIPAMPool structure.
type CreateIPAMPoolReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *CreateIPAMPoolReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 201:
		result := NewCreateIPAMPoolCreated()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewCreateIPAMPoolUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewCreateIPAMPoolForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewCreateIPAMPoolDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewCreateIPAMPoolCreated creates a CreateIPAMPoolCreated with default headers values
func NewCreateIPAMPoolCreated() *CreateIPAMPoolCreated {
	return &CreateIPAMPoolCreated{}
}

/*
CreateIPAMPoolCreated describes a response with status code 201, with default header values.

EmptyResponse is a empty response
*/
type CreateIPAMPoolCreated struct {
}

// IsSuccess returns true when this create Ip a m pool created response has a 2xx status code
func (o *CreateIPAMPoolCreated) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this create Ip a m pool created response has a 3xx status code
func (o *CreateIPAMPoolCreated) IsRedirect() bool {
	return false
}

// IsClientError returns true when this create Ip a m pool created response has a 4xx status code
func (o *CreateIPAMPoolCreated) IsClientError() bool {
	return false
}

// IsServerError returns true when this create Ip a m pool created response has a 5xx status code
func (o *CreateIPAMPoolCreated) IsServerError() bool {
	return false
}

// IsCode returns true when this create Ip a m pool created response a status code equal to that given
func (o *CreateIPAMPoolCreated) IsCode(code int) bool {
	return code == 201
}

func (o *CreateIPAMPoolCreated) Error() string {
	return fmt.Sprintf("[POST /api/v2/seeds/{seed_name}/ipampools][%d] createIpAMPoolCreated ", 201)
}

func (o *CreateIPAMPoolCreated) String() string {
	return fmt.Sprintf("[POST /api/v2/seeds/{seed_name}/ipampools][%d] createIpAMPoolCreated ", 201)
}

func (o *CreateIPAMPoolCreated) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateIPAMPoolUnauthorized creates a CreateIPAMPoolUnauthorized with default headers values
func NewCreateIPAMPoolUnauthorized() *CreateIPAMPoolUnauthorized {
	return &CreateIPAMPoolUnauthorized{}
}

/*
CreateIPAMPoolUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type CreateIPAMPoolUnauthorized struct {
}

// IsSuccess returns true when this create Ip a m pool unauthorized response has a 2xx status code
func (o *CreateIPAMPoolUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this create Ip a m pool unauthorized response has a 3xx status code
func (o *CreateIPAMPoolUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this create Ip a m pool unauthorized response has a 4xx status code
func (o *CreateIPAMPoolUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this create Ip a m pool unauthorized response has a 5xx status code
func (o *CreateIPAMPoolUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this create Ip a m pool unauthorized response a status code equal to that given
func (o *CreateIPAMPoolUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *CreateIPAMPoolUnauthorized) Error() string {
	return fmt.Sprintf("[POST /api/v2/seeds/{seed_name}/ipampools][%d] createIpAMPoolUnauthorized ", 401)
}

func (o *CreateIPAMPoolUnauthorized) String() string {
	return fmt.Sprintf("[POST /api/v2/seeds/{seed_name}/ipampools][%d] createIpAMPoolUnauthorized ", 401)
}

func (o *CreateIPAMPoolUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateIPAMPoolForbidden creates a CreateIPAMPoolForbidden with default headers values
func NewCreateIPAMPoolForbidden() *CreateIPAMPoolForbidden {
	return &CreateIPAMPoolForbidden{}
}

/*
CreateIPAMPoolForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type CreateIPAMPoolForbidden struct {
}

// IsSuccess returns true when this create Ip a m pool forbidden response has a 2xx status code
func (o *CreateIPAMPoolForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this create Ip a m pool forbidden response has a 3xx status code
func (o *CreateIPAMPoolForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this create Ip a m pool forbidden response has a 4xx status code
func (o *CreateIPAMPoolForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this create Ip a m pool forbidden response has a 5xx status code
func (o *CreateIPAMPoolForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this create Ip a m pool forbidden response a status code equal to that given
func (o *CreateIPAMPoolForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *CreateIPAMPoolForbidden) Error() string {
	return fmt.Sprintf("[POST /api/v2/seeds/{seed_name}/ipampools][%d] createIpAMPoolForbidden ", 403)
}

func (o *CreateIPAMPoolForbidden) String() string {
	return fmt.Sprintf("[POST /api/v2/seeds/{seed_name}/ipampools][%d] createIpAMPoolForbidden ", 403)
}

func (o *CreateIPAMPoolForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateIPAMPoolDefault creates a CreateIPAMPoolDefault with default headers values
func NewCreateIPAMPoolDefault(code int) *CreateIPAMPoolDefault {
	return &CreateIPAMPoolDefault{
		_statusCode: code,
	}
}

/*
CreateIPAMPoolDefault describes a response with status code -1, with default header values.

errorResponse
*/
type CreateIPAMPoolDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the create IP a m pool default response
func (o *CreateIPAMPoolDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this create IP a m pool default response has a 2xx status code
func (o *CreateIPAMPoolDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this create IP a m pool default response has a 3xx status code
func (o *CreateIPAMPoolDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this create IP a m pool default response has a 4xx status code
func (o *CreateIPAMPoolDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this create IP a m pool default response has a 5xx status code
func (o *CreateIPAMPoolDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this create IP a m pool default response a status code equal to that given
func (o *CreateIPAMPoolDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *CreateIPAMPoolDefault) Error() string {
	return fmt.Sprintf("[POST /api/v2/seeds/{seed_name}/ipampools][%d] createIPAMPool default  %+v", o._statusCode, o.Payload)
}

func (o *CreateIPAMPoolDefault) String() string {
	return fmt.Sprintf("[POST /api/v2/seeds/{seed_name}/ipampools][%d] createIPAMPool default  %+v", o._statusCode, o.Payload)
}

func (o *CreateIPAMPoolDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *CreateIPAMPoolDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
