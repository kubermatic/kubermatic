// Code generated by go-swagger; DO NOT EDIT.

package openstack

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListOpenstackSizesReader is a Reader for the ListOpenstackSizes structure.
type ListOpenstackSizesReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListOpenstackSizesReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListOpenstackSizesOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListOpenstackSizesDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListOpenstackSizesOK creates a ListOpenstackSizesOK with default headers values
func NewListOpenstackSizesOK() *ListOpenstackSizesOK {
	return &ListOpenstackSizesOK{}
}

/* ListOpenstackSizesOK describes a response with status code 200, with default header values.

OpenstackSize
*/
type ListOpenstackSizesOK struct {
	Payload []*models.OpenstackSize
}

// IsSuccess returns true when this list openstack sizes o k response has a 2xx status code
func (o *ListOpenstackSizesOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list openstack sizes o k response has a 3xx status code
func (o *ListOpenstackSizesOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list openstack sizes o k response has a 4xx status code
func (o *ListOpenstackSizesOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list openstack sizes o k response has a 5xx status code
func (o *ListOpenstackSizesOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list openstack sizes o k response a status code equal to that given
func (o *ListOpenstackSizesOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListOpenstackSizesOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/providers/openstack/sizes][%d] listOpenstackSizesOK  %+v", 200, o.Payload)
}

func (o *ListOpenstackSizesOK) String() string {
	return fmt.Sprintf("[GET /api/v1/providers/openstack/sizes][%d] listOpenstackSizesOK  %+v", 200, o.Payload)
}

func (o *ListOpenstackSizesOK) GetPayload() []*models.OpenstackSize {
	return o.Payload
}

func (o *ListOpenstackSizesOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListOpenstackSizesDefault creates a ListOpenstackSizesDefault with default headers values
func NewListOpenstackSizesDefault(code int) *ListOpenstackSizesDefault {
	return &ListOpenstackSizesDefault{
		_statusCode: code,
	}
}

/* ListOpenstackSizesDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListOpenstackSizesDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list openstack sizes default response
func (o *ListOpenstackSizesDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list openstack sizes default response has a 2xx status code
func (o *ListOpenstackSizesDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list openstack sizes default response has a 3xx status code
func (o *ListOpenstackSizesDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list openstack sizes default response has a 4xx status code
func (o *ListOpenstackSizesDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list openstack sizes default response has a 5xx status code
func (o *ListOpenstackSizesDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list openstack sizes default response a status code equal to that given
func (o *ListOpenstackSizesDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListOpenstackSizesDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/providers/openstack/sizes][%d] listOpenstackSizes default  %+v", o._statusCode, o.Payload)
}

func (o *ListOpenstackSizesDefault) String() string {
	return fmt.Sprintf("[GET /api/v1/providers/openstack/sizes][%d] listOpenstackSizes default  %+v", o._statusCode, o.Payload)
}

func (o *ListOpenstackSizesDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListOpenstackSizesDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
