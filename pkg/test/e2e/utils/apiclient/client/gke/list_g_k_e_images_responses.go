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

// ListGKEImagesReader is a Reader for the ListGKEImages structure.
type ListGKEImagesReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListGKEImagesReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListGKEImagesOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListGKEImagesDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListGKEImagesOK creates a ListGKEImagesOK with default headers values
func NewListGKEImagesOK() *ListGKEImagesOK {
	return &ListGKEImagesOK{}
}

/*
ListGKEImagesOK describes a response with status code 200, with default header values.

GKEImageList
*/
type ListGKEImagesOK struct {
	Payload models.GKEImageList
}

// IsSuccess returns true when this list g k e images o k response has a 2xx status code
func (o *ListGKEImagesOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list g k e images o k response has a 3xx status code
func (o *ListGKEImagesOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list g k e images o k response has a 4xx status code
func (o *ListGKEImagesOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list g k e images o k response has a 5xx status code
func (o *ListGKEImagesOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list g k e images o k response a status code equal to that given
func (o *ListGKEImagesOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListGKEImagesOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/gke/images][%d] listGKEImagesOK  %+v", 200, o.Payload)
}

func (o *ListGKEImagesOK) String() string {
	return fmt.Sprintf("[GET /api/v2/providers/gke/images][%d] listGKEImagesOK  %+v", 200, o.Payload)
}

func (o *ListGKEImagesOK) GetPayload() models.GKEImageList {
	return o.Payload
}

func (o *ListGKEImagesOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListGKEImagesDefault creates a ListGKEImagesDefault with default headers values
func NewListGKEImagesDefault(code int) *ListGKEImagesDefault {
	return &ListGKEImagesDefault{
		_statusCode: code,
	}
}

/*
ListGKEImagesDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListGKEImagesDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list g k e images default response
func (o *ListGKEImagesDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list g k e images default response has a 2xx status code
func (o *ListGKEImagesDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list g k e images default response has a 3xx status code
func (o *ListGKEImagesDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list g k e images default response has a 4xx status code
func (o *ListGKEImagesDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list g k e images default response has a 5xx status code
func (o *ListGKEImagesDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list g k e images default response a status code equal to that given
func (o *ListGKEImagesDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListGKEImagesDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/gke/images][%d] listGKEImages default  %+v", o._statusCode, o.Payload)
}

func (o *ListGKEImagesDefault) String() string {
	return fmt.Sprintf("[GET /api/v2/providers/gke/images][%d] listGKEImages default  %+v", o._statusCode, o.Payload)
}

func (o *ListGKEImagesDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListGKEImagesDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
