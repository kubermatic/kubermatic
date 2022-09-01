// Code generated by go-swagger; DO NOT EDIT.

package nutanix

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListNutanixCategoryValuesReader is a Reader for the ListNutanixCategoryValues structure.
type ListNutanixCategoryValuesReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListNutanixCategoryValuesReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListNutanixCategoryValuesOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListNutanixCategoryValuesDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListNutanixCategoryValuesOK creates a ListNutanixCategoryValuesOK with default headers values
func NewListNutanixCategoryValuesOK() *ListNutanixCategoryValuesOK {
	return &ListNutanixCategoryValuesOK{}
}

/* ListNutanixCategoryValuesOK describes a response with status code 200, with default header values.

NutanixCategoryValueList
*/
type ListNutanixCategoryValuesOK struct {
	Payload models.NutanixCategoryValueList
}

// IsSuccess returns true when this list nutanix category values o k response has a 2xx status code
func (o *ListNutanixCategoryValuesOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list nutanix category values o k response has a 3xx status code
func (o *ListNutanixCategoryValuesOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list nutanix category values o k response has a 4xx status code
func (o *ListNutanixCategoryValuesOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list nutanix category values o k response has a 5xx status code
func (o *ListNutanixCategoryValuesOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list nutanix category values o k response a status code equal to that given
func (o *ListNutanixCategoryValuesOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListNutanixCategoryValuesOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/nutanix/{dc}/categories/{category}/values][%d] listNutanixCategoryValuesOK  %+v", 200, o.Payload)
}

func (o *ListNutanixCategoryValuesOK) String() string {
	return fmt.Sprintf("[GET /api/v2/providers/nutanix/{dc}/categories/{category}/values][%d] listNutanixCategoryValuesOK  %+v", 200, o.Payload)
}

func (o *ListNutanixCategoryValuesOK) GetPayload() models.NutanixCategoryValueList {
	return o.Payload
}

func (o *ListNutanixCategoryValuesOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListNutanixCategoryValuesDefault creates a ListNutanixCategoryValuesDefault with default headers values
func NewListNutanixCategoryValuesDefault(code int) *ListNutanixCategoryValuesDefault {
	return &ListNutanixCategoryValuesDefault{
		_statusCode: code,
	}
}

/* ListNutanixCategoryValuesDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListNutanixCategoryValuesDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list nutanix category values default response
func (o *ListNutanixCategoryValuesDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list nutanix category values default response has a 2xx status code
func (o *ListNutanixCategoryValuesDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list nutanix category values default response has a 3xx status code
func (o *ListNutanixCategoryValuesDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list nutanix category values default response has a 4xx status code
func (o *ListNutanixCategoryValuesDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list nutanix category values default response has a 5xx status code
func (o *ListNutanixCategoryValuesDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list nutanix category values default response a status code equal to that given
func (o *ListNutanixCategoryValuesDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListNutanixCategoryValuesDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/nutanix/{dc}/categories/{category}/values][%d] listNutanixCategoryValues default  %+v", o._statusCode, o.Payload)
}

func (o *ListNutanixCategoryValuesDefault) String() string {
	return fmt.Sprintf("[GET /api/v2/providers/nutanix/{dc}/categories/{category}/values][%d] listNutanixCategoryValues default  %+v", o._statusCode, o.Payload)
}

func (o *ListNutanixCategoryValuesDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListNutanixCategoryValuesDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
