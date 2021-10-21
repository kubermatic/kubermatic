// Code generated by go-swagger; DO NOT EDIT.

package regions

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListEC2RegionsReader is a Reader for the ListEC2Regions structure.
type ListEC2RegionsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListEC2RegionsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListEC2RegionsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListEC2RegionsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListEC2RegionsForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListEC2RegionsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListEC2RegionsOK creates a ListEC2RegionsOK with default headers values
func NewListEC2RegionsOK() *ListEC2RegionsOK {
	return &ListEC2RegionsOK{}
}

/* ListEC2RegionsOK describes a response with status code 200, with default header values.

Regions
*/
type ListEC2RegionsOK struct {
	Payload []models.Regions
}

func (o *ListEC2RegionsOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/eks/regions][%d] listEC2RegionsOK  %+v", 200, o.Payload)
}
func (o *ListEC2RegionsOK) GetPayload() []models.Regions {
	return o.Payload
}

func (o *ListEC2RegionsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListEC2RegionsUnauthorized creates a ListEC2RegionsUnauthorized with default headers values
func NewListEC2RegionsUnauthorized() *ListEC2RegionsUnauthorized {
	return &ListEC2RegionsUnauthorized{}
}

/* ListEC2RegionsUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type ListEC2RegionsUnauthorized struct {
}

func (o *ListEC2RegionsUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/eks/regions][%d] listEC2RegionsUnauthorized ", 401)
}

func (o *ListEC2RegionsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListEC2RegionsForbidden creates a ListEC2RegionsForbidden with default headers values
func NewListEC2RegionsForbidden() *ListEC2RegionsForbidden {
	return &ListEC2RegionsForbidden{}
}

/* ListEC2RegionsForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type ListEC2RegionsForbidden struct {
}

func (o *ListEC2RegionsForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/eks/regions][%d] listEC2RegionsForbidden ", 403)
}

func (o *ListEC2RegionsForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListEC2RegionsDefault creates a ListEC2RegionsDefault with default headers values
func NewListEC2RegionsDefault(code int) *ListEC2RegionsDefault {
	return &ListEC2RegionsDefault{
		_statusCode: code,
	}
}

/* ListEC2RegionsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListEC2RegionsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list e c2 regions default response
func (o *ListEC2RegionsDefault) Code() int {
	return o._statusCode
}

func (o *ListEC2RegionsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/eks/regions][%d] listEC2Regions default  %+v", o._statusCode, o.Payload)
}
func (o *ListEC2RegionsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListEC2RegionsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
