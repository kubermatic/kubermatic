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

// ListAzureSubnetsReader is a Reader for the ListAzureSubnets structure.
type ListAzureSubnetsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListAzureSubnetsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListAzureSubnetsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListAzureSubnetsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListAzureSubnetsOK creates a ListAzureSubnetsOK with default headers values
func NewListAzureSubnetsOK() *ListAzureSubnetsOK {
	return &ListAzureSubnetsOK{}
}

/*ListAzureSubnetsOK handles this case with default header values.

AzureSubnetsList
*/
type ListAzureSubnetsOK struct {
	Payload *models.AzureSubnetsList
}

func (o *ListAzureSubnetsOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/azure/subnets][%d] listAzureSubnetsOK  %+v", 200, o.Payload)
}

func (o *ListAzureSubnetsOK) GetPayload() *models.AzureSubnetsList {
	return o.Payload
}

func (o *ListAzureSubnetsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.AzureSubnetsList)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListAzureSubnetsDefault creates a ListAzureSubnetsDefault with default headers values
func NewListAzureSubnetsDefault(code int) *ListAzureSubnetsDefault {
	return &ListAzureSubnetsDefault{
		_statusCode: code,
	}
}

/*ListAzureSubnetsDefault handles this case with default header values.

errorResponse
*/
type ListAzureSubnetsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list azure subnets default response
func (o *ListAzureSubnetsDefault) Code() int {
	return o._statusCode
}

func (o *ListAzureSubnetsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/azure/subnets][%d] listAzureSubnets default  %+v", o._statusCode, o.Payload)
}

func (o *ListAzureSubnetsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListAzureSubnetsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
