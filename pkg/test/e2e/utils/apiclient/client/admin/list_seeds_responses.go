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

// ListSeedsReader is a Reader for the ListSeeds structure.
type ListSeedsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListSeedsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListSeedsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListSeedsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListSeedsForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListSeedsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListSeedsOK creates a ListSeedsOK with default headers values
func NewListSeedsOK() *ListSeedsOK {
	return &ListSeedsOK{}
}

/*
ListSeedsOK describes a response with status code 200, with default header values.

Seed
*/
type ListSeedsOK struct {
	Payload []*models.Seed
}

// IsSuccess returns true when this list seeds o k response has a 2xx status code
func (o *ListSeedsOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list seeds o k response has a 3xx status code
func (o *ListSeedsOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list seeds o k response has a 4xx status code
func (o *ListSeedsOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list seeds o k response has a 5xx status code
func (o *ListSeedsOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list seeds o k response a status code equal to that given
func (o *ListSeedsOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListSeedsOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin/seeds][%d] listSeedsOK  %+v", 200, o.Payload)
}

func (o *ListSeedsOK) String() string {
	return fmt.Sprintf("[GET /api/v1/admin/seeds][%d] listSeedsOK  %+v", 200, o.Payload)
}

func (o *ListSeedsOK) GetPayload() []*models.Seed {
	return o.Payload
}

func (o *ListSeedsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListSeedsUnauthorized creates a ListSeedsUnauthorized with default headers values
func NewListSeedsUnauthorized() *ListSeedsUnauthorized {
	return &ListSeedsUnauthorized{}
}

/*
ListSeedsUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type ListSeedsUnauthorized struct {
}

// IsSuccess returns true when this list seeds unauthorized response has a 2xx status code
func (o *ListSeedsUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list seeds unauthorized response has a 3xx status code
func (o *ListSeedsUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list seeds unauthorized response has a 4xx status code
func (o *ListSeedsUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this list seeds unauthorized response has a 5xx status code
func (o *ListSeedsUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this list seeds unauthorized response a status code equal to that given
func (o *ListSeedsUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *ListSeedsUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin/seeds][%d] listSeedsUnauthorized ", 401)
}

func (o *ListSeedsUnauthorized) String() string {
	return fmt.Sprintf("[GET /api/v1/admin/seeds][%d] listSeedsUnauthorized ", 401)
}

func (o *ListSeedsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListSeedsForbidden creates a ListSeedsForbidden with default headers values
func NewListSeedsForbidden() *ListSeedsForbidden {
	return &ListSeedsForbidden{}
}

/*
ListSeedsForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type ListSeedsForbidden struct {
}

// IsSuccess returns true when this list seeds forbidden response has a 2xx status code
func (o *ListSeedsForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list seeds forbidden response has a 3xx status code
func (o *ListSeedsForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list seeds forbidden response has a 4xx status code
func (o *ListSeedsForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this list seeds forbidden response has a 5xx status code
func (o *ListSeedsForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this list seeds forbidden response a status code equal to that given
func (o *ListSeedsForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *ListSeedsForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin/seeds][%d] listSeedsForbidden ", 403)
}

func (o *ListSeedsForbidden) String() string {
	return fmt.Sprintf("[GET /api/v1/admin/seeds][%d] listSeedsForbidden ", 403)
}

func (o *ListSeedsForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListSeedsDefault creates a ListSeedsDefault with default headers values
func NewListSeedsDefault(code int) *ListSeedsDefault {
	return &ListSeedsDefault{
		_statusCode: code,
	}
}

/*
ListSeedsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListSeedsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list seeds default response
func (o *ListSeedsDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list seeds default response has a 2xx status code
func (o *ListSeedsDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list seeds default response has a 3xx status code
func (o *ListSeedsDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list seeds default response has a 4xx status code
func (o *ListSeedsDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list seeds default response has a 5xx status code
func (o *ListSeedsDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list seeds default response a status code equal to that given
func (o *ListSeedsDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListSeedsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin/seeds][%d] listSeeds default  %+v", o._statusCode, o.Payload)
}

func (o *ListSeedsDefault) String() string {
	return fmt.Sprintf("[GET /api/v1/admin/seeds][%d] listSeeds default  %+v", o._statusCode, o.Payload)
}

func (o *ListSeedsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListSeedsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
