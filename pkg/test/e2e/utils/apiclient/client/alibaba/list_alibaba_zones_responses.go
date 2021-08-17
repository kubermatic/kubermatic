// Code generated by go-swagger; DO NOT EDIT.

package alibaba

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListAlibabaZonesReader is a Reader for the ListAlibabaZones structure.
type ListAlibabaZonesReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListAlibabaZonesReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListAlibabaZonesOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListAlibabaZonesDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListAlibabaZonesOK creates a ListAlibabaZonesOK with default headers values
func NewListAlibabaZonesOK() *ListAlibabaZonesOK {
	return &ListAlibabaZonesOK{}
}

/* ListAlibabaZonesOK describes a response with status code 200, with default header values.

AlibabaZoneList
*/
type ListAlibabaZonesOK struct {
	Payload models.AlibabaZoneList
}

func (o *ListAlibabaZonesOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/providers/alibaba/zones][%d] listAlibabaZonesOK  %+v", 200, o.Payload)
}
func (o *ListAlibabaZonesOK) GetPayload() models.AlibabaZoneList {
	return o.Payload
}

func (o *ListAlibabaZonesOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListAlibabaZonesDefault creates a ListAlibabaZonesDefault with default headers values
func NewListAlibabaZonesDefault(code int) *ListAlibabaZonesDefault {
	return &ListAlibabaZonesDefault{
		_statusCode: code,
	}
}

/* ListAlibabaZonesDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListAlibabaZonesDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list alibaba zones default response
func (o *ListAlibabaZonesDefault) Code() int {
	return o._statusCode
}

func (o *ListAlibabaZonesDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/providers/alibaba/zones][%d] listAlibabaZones default  %+v", o._statusCode, o.Payload)
}
func (o *ListAlibabaZonesDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListAlibabaZonesDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
