// Code generated by go-swagger; DO NOT EDIT.

package aws

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// ListAWSSubnetsReader is a Reader for the ListAWSSubnets structure.
type ListAWSSubnetsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListAWSSubnetsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListAWSSubnetsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListAWSSubnetsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListAWSSubnetsOK creates a ListAWSSubnetsOK with default headers values
func NewListAWSSubnetsOK() *ListAWSSubnetsOK {
	return &ListAWSSubnetsOK{}
}

/*ListAWSSubnetsOK handles this case with default header values.

AWSSubnetList
*/
type ListAWSSubnetsOK struct {
	Payload models.AWSSubnetList
}

func (o *ListAWSSubnetsOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/providers/aws/{dc}/subnets][%d] listAWSSubnetsOK  %+v", 200, o.Payload)
}

func (o *ListAWSSubnetsOK) GetPayload() models.AWSSubnetList {
	return o.Payload
}

func (o *ListAWSSubnetsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListAWSSubnetsDefault creates a ListAWSSubnetsDefault with default headers values
func NewListAWSSubnetsDefault(code int) *ListAWSSubnetsDefault {
	return &ListAWSSubnetsDefault{
		_statusCode: code,
	}
}

/*ListAWSSubnetsDefault handles this case with default header values.

errorResponse
*/
type ListAWSSubnetsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list a w s subnets default response
func (o *ListAWSSubnetsDefault) Code() int {
	return o._statusCode
}

func (o *ListAWSSubnetsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/providers/aws/{dc}/subnets][%d] listAWSSubnets default  %+v", o._statusCode, o.Payload)
}

func (o *ListAWSSubnetsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListAWSSubnetsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
