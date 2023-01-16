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

// ListOpenstackSubnetPoolsReader is a Reader for the ListOpenstackSubnetPools structure.
type ListOpenstackSubnetPoolsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListOpenstackSubnetPoolsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListOpenstackSubnetPoolsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListOpenstackSubnetPoolsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListOpenstackSubnetPoolsOK creates a ListOpenstackSubnetPoolsOK with default headers values
func NewListOpenstackSubnetPoolsOK() *ListOpenstackSubnetPoolsOK {
	return &ListOpenstackSubnetPoolsOK{}
}

/* ListOpenstackSubnetPoolsOK describes a response with status code 200, with default header values.

OpenstackSubnetPool
*/
type ListOpenstackSubnetPoolsOK struct {
	Payload []*models.OpenstackSubnetPool
}

func (o *ListOpenstackSubnetPoolsOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/openstack/subnetpools][%d] listOpenstackSubnetPoolsOK  %+v", 200, o.Payload)
}
func (o *ListOpenstackSubnetPoolsOK) GetPayload() []*models.OpenstackSubnetPool {
	return o.Payload
}

func (o *ListOpenstackSubnetPoolsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListOpenstackSubnetPoolsDefault creates a ListOpenstackSubnetPoolsDefault with default headers values
func NewListOpenstackSubnetPoolsDefault(code int) *ListOpenstackSubnetPoolsDefault {
	return &ListOpenstackSubnetPoolsDefault{
		_statusCode: code,
	}
}

/* ListOpenstackSubnetPoolsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListOpenstackSubnetPoolsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list openstack subnet pools default response
func (o *ListOpenstackSubnetPoolsDefault) Code() int {
	return o._statusCode
}

func (o *ListOpenstackSubnetPoolsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/openstack/subnetpools][%d] listOpenstackSubnetPools default  %+v", o._statusCode, o.Payload)
}
func (o *ListOpenstackSubnetPoolsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListOpenstackSubnetPoolsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}