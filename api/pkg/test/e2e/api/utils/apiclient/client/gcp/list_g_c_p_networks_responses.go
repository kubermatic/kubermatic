// Code generated by go-swagger; DO NOT EDIT.

package gcp

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// ListGCPNetworksReader is a Reader for the ListGCPNetworks structure.
type ListGCPNetworksReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListGCPNetworksReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListGCPNetworksOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListGCPNetworksDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListGCPNetworksOK creates a ListGCPNetworksOK with default headers values
func NewListGCPNetworksOK() *ListGCPNetworksOK {
	return &ListGCPNetworksOK{}
}

/*ListGCPNetworksOK handles this case with default header values.

GCPNetworkList
*/
type ListGCPNetworksOK struct {
	Payload models.GCPNetworkList
}

func (o *ListGCPNetworksOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/providers/gcp/networks][%d] listGCPNetworksOK  %+v", 200, o.Payload)
}

func (o *ListGCPNetworksOK) GetPayload() models.GCPNetworkList {
	return o.Payload
}

func (o *ListGCPNetworksOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListGCPNetworksDefault creates a ListGCPNetworksDefault with default headers values
func NewListGCPNetworksDefault(code int) *ListGCPNetworksDefault {
	return &ListGCPNetworksDefault{
		_statusCode: code,
	}
}

/*ListGCPNetworksDefault handles this case with default header values.

errorResponse
*/
type ListGCPNetworksDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list g c p networks default response
func (o *ListGCPNetworksDefault) Code() int {
	return o._statusCode
}

func (o *ListGCPNetworksDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/providers/gcp/networks][%d] listGCPNetworks default  %+v", o._statusCode, o.Payload)
}

func (o *ListGCPNetworksDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListGCPNetworksDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
