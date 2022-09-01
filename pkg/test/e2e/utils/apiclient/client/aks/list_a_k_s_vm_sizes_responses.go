// Code generated by go-swagger; DO NOT EDIT.

package aks

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListAKSVMSizesReader is a Reader for the ListAKSVMSizes structure.
type ListAKSVMSizesReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListAKSVMSizesReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListAKSVMSizesOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListAKSVMSizesDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListAKSVMSizesOK creates a ListAKSVMSizesOK with default headers values
func NewListAKSVMSizesOK() *ListAKSVMSizesOK {
	return &ListAKSVMSizesOK{}
}

/* ListAKSVMSizesOK describes a response with status code 200, with default header values.

AKSVMSizeList
*/
type ListAKSVMSizesOK struct {
	Payload models.AKSVMSizeList
}

// IsSuccess returns true when this list a k s Vm sizes o k response has a 2xx status code
func (o *ListAKSVMSizesOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list a k s Vm sizes o k response has a 3xx status code
func (o *ListAKSVMSizesOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list a k s Vm sizes o k response has a 4xx status code
func (o *ListAKSVMSizesOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list a k s Vm sizes o k response has a 5xx status code
func (o *ListAKSVMSizesOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list a k s Vm sizes o k response a status code equal to that given
func (o *ListAKSVMSizesOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListAKSVMSizesOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/aks/vmsizes][%d] listAKSVmSizesOK  %+v", 200, o.Payload)
}

func (o *ListAKSVMSizesOK) String() string {
	return fmt.Sprintf("[GET /api/v2/providers/aks/vmsizes][%d] listAKSVmSizesOK  %+v", 200, o.Payload)
}

func (o *ListAKSVMSizesOK) GetPayload() models.AKSVMSizeList {
	return o.Payload
}

func (o *ListAKSVMSizesOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListAKSVMSizesDefault creates a ListAKSVMSizesDefault with default headers values
func NewListAKSVMSizesDefault(code int) *ListAKSVMSizesDefault {
	return &ListAKSVMSizesDefault{
		_statusCode: code,
	}
}

/* ListAKSVMSizesDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListAKSVMSizesDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list a k s VM sizes default response
func (o *ListAKSVMSizesDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list a k s VM sizes default response has a 2xx status code
func (o *ListAKSVMSizesDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list a k s VM sizes default response has a 3xx status code
func (o *ListAKSVMSizesDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list a k s VM sizes default response has a 4xx status code
func (o *ListAKSVMSizesDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list a k s VM sizes default response has a 5xx status code
func (o *ListAKSVMSizesDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list a k s VM sizes default response a status code equal to that given
func (o *ListAKSVMSizesDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListAKSVMSizesDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/aks/vmsizes][%d] listAKSVMSizes default  %+v", o._statusCode, o.Payload)
}

func (o *ListAKSVMSizesDefault) String() string {
	return fmt.Sprintf("[GET /api/v2/providers/aks/vmsizes][%d] listAKSVMSizes default  %+v", o._statusCode, o.Payload)
}

func (o *ListAKSVMSizesDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListAKSVMSizesDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
