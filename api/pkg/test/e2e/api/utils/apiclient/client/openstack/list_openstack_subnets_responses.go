// Code generated by go-swagger; DO NOT EDIT.

package openstack

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	models "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// ListOpenstackSubnetsReader is a Reader for the ListOpenstackSubnets structure.
type ListOpenstackSubnetsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListOpenstackSubnetsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewListOpenstackSubnetsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	default:
		result := NewListOpenstackSubnetsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListOpenstackSubnetsOK creates a ListOpenstackSubnetsOK with default headers values
func NewListOpenstackSubnetsOK() *ListOpenstackSubnetsOK {
	return &ListOpenstackSubnetsOK{}
}

/*ListOpenstackSubnetsOK handles this case with default header values.

OpenstackSubnet
*/
type ListOpenstackSubnetsOK struct {
	Payload []*models.OpenstackSubnet
}

func (o *ListOpenstackSubnetsOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/providers/openstack/subnets][%d] listOpenstackSubnetsOK  %+v", 200, o.Payload)
}

func (o *ListOpenstackSubnetsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListOpenstackSubnetsDefault creates a ListOpenstackSubnetsDefault with default headers values
func NewListOpenstackSubnetsDefault(code int) *ListOpenstackSubnetsDefault {
	return &ListOpenstackSubnetsDefault{
		_statusCode: code,
	}
}

/*ListOpenstackSubnetsDefault handles this case with default header values.

ErrorResponse is the default representation of an error
*/
type ListOpenstackSubnetsDefault struct {
	_statusCode int

	Payload *models.ErrorDetails
}

// Code gets the status code for the list openstack subnets default response
func (o *ListOpenstackSubnetsDefault) Code() int {
	return o._statusCode
}

func (o *ListOpenstackSubnetsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/providers/openstack/subnets][%d] listOpenstackSubnets default  %+v", o._statusCode, o.Payload)
}

func (o *ListOpenstackSubnetsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorDetails)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
