// Code generated by go-swagger; DO NOT EDIT.

package gke

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListGKEDiskTypesReader is a Reader for the ListGKEDiskTypes structure.
type ListGKEDiskTypesReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListGKEDiskTypesReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListGKEDiskTypesOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListGKEDiskTypesDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListGKEDiskTypesOK creates a ListGKEDiskTypesOK with default headers values
func NewListGKEDiskTypesOK() *ListGKEDiskTypesOK {
	return &ListGKEDiskTypesOK{}
}

/* ListGKEDiskTypesOK describes a response with status code 200, with default header values.

GKEDiskTypeList
*/
type ListGKEDiskTypesOK struct {
	Payload models.GKEDiskTypeList
}

// IsSuccess returns true when this list g k e disk types o k response has a 2xx status code
func (o *ListGKEDiskTypesOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list g k e disk types o k response has a 3xx status code
func (o *ListGKEDiskTypesOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list g k e disk types o k response has a 4xx status code
func (o *ListGKEDiskTypesOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list g k e disk types o k response has a 5xx status code
func (o *ListGKEDiskTypesOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list g k e disk types o k response a status code equal to that given
func (o *ListGKEDiskTypesOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListGKEDiskTypesOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/gke/disktypes][%d] listGKEDiskTypesOK  %+v", 200, o.Payload)
}

func (o *ListGKEDiskTypesOK) String() string {
	return fmt.Sprintf("[GET /api/v2/providers/gke/disktypes][%d] listGKEDiskTypesOK  %+v", 200, o.Payload)
}

func (o *ListGKEDiskTypesOK) GetPayload() models.GKEDiskTypeList {
	return o.Payload
}

func (o *ListGKEDiskTypesOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListGKEDiskTypesDefault creates a ListGKEDiskTypesDefault with default headers values
func NewListGKEDiskTypesDefault(code int) *ListGKEDiskTypesDefault {
	return &ListGKEDiskTypesDefault{
		_statusCode: code,
	}
}

/* ListGKEDiskTypesDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListGKEDiskTypesDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list g k e disk types default response
func (o *ListGKEDiskTypesDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list g k e disk types default response has a 2xx status code
func (o *ListGKEDiskTypesDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list g k e disk types default response has a 3xx status code
func (o *ListGKEDiskTypesDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list g k e disk types default response has a 4xx status code
func (o *ListGKEDiskTypesDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list g k e disk types default response has a 5xx status code
func (o *ListGKEDiskTypesDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list g k e disk types default response a status code equal to that given
func (o *ListGKEDiskTypesDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListGKEDiskTypesDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/gke/disktypes][%d] listGKEDiskTypes default  %+v", o._statusCode, o.Payload)
}

func (o *ListGKEDiskTypesDefault) String() string {
	return fmt.Sprintf("[GET /api/v2/providers/gke/disktypes][%d] listGKEDiskTypes default  %+v", o._statusCode, o.Payload)
}

func (o *ListGKEDiskTypesDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListGKEDiskTypesDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
