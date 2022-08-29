// Code generated by go-swagger; DO NOT EDIT.

package aws

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListAWSVPCSReader is a Reader for the ListAWSVPCS structure.
type ListAWSVPCSReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListAWSVPCSReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListAWSVPCSOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListAWSVPCSDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListAWSVPCSOK creates a ListAWSVPCSOK with default headers values
func NewListAWSVPCSOK() *ListAWSVPCSOK {
	return &ListAWSVPCSOK{}
}

/*
ListAWSVPCSOK describes a response with status code 200, with default header values.

AWSVPCList
*/
type ListAWSVPCSOK struct {
	Payload models.AWSVPCList
}

// IsSuccess returns true when this list a w s v p c s o k response has a 2xx status code
func (o *ListAWSVPCSOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list a w s v p c s o k response has a 3xx status code
func (o *ListAWSVPCSOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list a w s v p c s o k response has a 4xx status code
func (o *ListAWSVPCSOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list a w s v p c s o k response has a 5xx status code
func (o *ListAWSVPCSOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list a w s v p c s o k response a status code equal to that given
func (o *ListAWSVPCSOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListAWSVPCSOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/providers/aws/{dc}/vpcs][%d] listAWSVPCSOK  %+v", 200, o.Payload)
}

func (o *ListAWSVPCSOK) String() string {
	return fmt.Sprintf("[GET /api/v1/providers/aws/{dc}/vpcs][%d] listAWSVPCSOK  %+v", 200, o.Payload)
}

func (o *ListAWSVPCSOK) GetPayload() models.AWSVPCList {
	return o.Payload
}

func (o *ListAWSVPCSOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListAWSVPCSDefault creates a ListAWSVPCSDefault with default headers values
func NewListAWSVPCSDefault(code int) *ListAWSVPCSDefault {
	return &ListAWSVPCSDefault{
		_statusCode: code,
	}
}

/*
ListAWSVPCSDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListAWSVPCSDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list a w s v p c s default response
func (o *ListAWSVPCSDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list a w s v p c s default response has a 2xx status code
func (o *ListAWSVPCSDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list a w s v p c s default response has a 3xx status code
func (o *ListAWSVPCSDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list a w s v p c s default response has a 4xx status code
func (o *ListAWSVPCSDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list a w s v p c s default response has a 5xx status code
func (o *ListAWSVPCSDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list a w s v p c s default response a status code equal to that given
func (o *ListAWSVPCSDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListAWSVPCSDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/providers/aws/{dc}/vpcs][%d] listAWSVPCS default  %+v", o._statusCode, o.Payload)
}

func (o *ListAWSVPCSDefault) String() string {
	return fmt.Sprintf("[GET /api/v1/providers/aws/{dc}/vpcs][%d] listAWSVPCS default  %+v", o._statusCode, o.Payload)
}

func (o *ListAWSVPCSDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListAWSVPCSDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
