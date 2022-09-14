// Code generated by go-swagger; DO NOT EDIT.

package azure

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListAzureRouteTablesReader is a Reader for the ListAzureRouteTables structure.
type ListAzureRouteTablesReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListAzureRouteTablesReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListAzureRouteTablesOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListAzureRouteTablesDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListAzureRouteTablesOK creates a ListAzureRouteTablesOK with default headers values
func NewListAzureRouteTablesOK() *ListAzureRouteTablesOK {
	return &ListAzureRouteTablesOK{}
}

/*
ListAzureRouteTablesOK describes a response with status code 200, with default header values.

AzureRouteTablesList
*/
type ListAzureRouteTablesOK struct {
	Payload *models.AzureRouteTablesList
}

// IsSuccess returns true when this list azure route tables o k response has a 2xx status code
func (o *ListAzureRouteTablesOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list azure route tables o k response has a 3xx status code
func (o *ListAzureRouteTablesOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list azure route tables o k response has a 4xx status code
func (o *ListAzureRouteTablesOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list azure route tables o k response has a 5xx status code
func (o *ListAzureRouteTablesOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list azure route tables o k response a status code equal to that given
func (o *ListAzureRouteTablesOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListAzureRouteTablesOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/azure/routetables][%d] listAzureRouteTablesOK  %+v", 200, o.Payload)
}

func (o *ListAzureRouteTablesOK) String() string {
	return fmt.Sprintf("[GET /api/v2/providers/azure/routetables][%d] listAzureRouteTablesOK  %+v", 200, o.Payload)
}

func (o *ListAzureRouteTablesOK) GetPayload() *models.AzureRouteTablesList {
	return o.Payload
}

func (o *ListAzureRouteTablesOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.AzureRouteTablesList)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListAzureRouteTablesDefault creates a ListAzureRouteTablesDefault with default headers values
func NewListAzureRouteTablesDefault(code int) *ListAzureRouteTablesDefault {
	return &ListAzureRouteTablesDefault{
		_statusCode: code,
	}
}

/*
ListAzureRouteTablesDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListAzureRouteTablesDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list azure route tables default response
func (o *ListAzureRouteTablesDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list azure route tables default response has a 2xx status code
func (o *ListAzureRouteTablesDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list azure route tables default response has a 3xx status code
func (o *ListAzureRouteTablesDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list azure route tables default response has a 4xx status code
func (o *ListAzureRouteTablesDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list azure route tables default response has a 5xx status code
func (o *ListAzureRouteTablesDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list azure route tables default response a status code equal to that given
func (o *ListAzureRouteTablesDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListAzureRouteTablesDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/azure/routetables][%d] listAzureRouteTables default  %+v", o._statusCode, o.Payload)
}

func (o *ListAzureRouteTablesDefault) String() string {
	return fmt.Sprintf("[GET /api/v2/providers/azure/routetables][%d] listAzureRouteTables default  %+v", o._statusCode, o.Payload)
}

func (o *ListAzureRouteTablesDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListAzureRouteTablesDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
