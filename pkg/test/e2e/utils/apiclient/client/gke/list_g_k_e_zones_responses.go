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

// ListGKEZonesReader is a Reader for the ListGKEZones structure.
type ListGKEZonesReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListGKEZonesReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListGKEZonesOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListGKEZonesDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListGKEZonesOK creates a ListGKEZonesOK with default headers values
func NewListGKEZonesOK() *ListGKEZonesOK {
	return &ListGKEZonesOK{}
}

/* ListGKEZonesOK describes a response with status code 200, with default header values.

GKEZoneList
*/
type ListGKEZonesOK struct {
	Payload models.GKEZoneList
}

func (o *ListGKEZonesOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/gke/zones][%d] listGKEZonesOK  %+v", 200, o.Payload)
}
func (o *ListGKEZonesOK) GetPayload() models.GKEZoneList {
	return o.Payload
}

func (o *ListGKEZonesOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListGKEZonesDefault creates a ListGKEZonesDefault with default headers values
func NewListGKEZonesDefault(code int) *ListGKEZonesDefault {
	return &ListGKEZonesDefault{
		_statusCode: code,
	}
}

/* ListGKEZonesDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListGKEZonesDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list g k e zones default response
func (o *ListGKEZonesDefault) Code() int {
	return o._statusCode
}

func (o *ListGKEZonesDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/gke/zones][%d] listGKEZones default  %+v", o._statusCode, o.Payload)
}
func (o *ListGKEZonesDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListGKEZonesDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
