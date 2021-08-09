// Code generated by go-swagger; DO NOT EDIT.

package anexia

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListAnexiaTemplatesReader is a Reader for the ListAnexiaTemplates structure.
type ListAnexiaTemplatesReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListAnexiaTemplatesReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListAnexiaTemplatesOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListAnexiaTemplatesDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListAnexiaTemplatesOK creates a ListAnexiaTemplatesOK with default headers values
func NewListAnexiaTemplatesOK() *ListAnexiaTemplatesOK {
	return &ListAnexiaTemplatesOK{}
}

/* ListAnexiaTemplatesOK describes a response with status code 200, with default header values.

AnexiaTemplateList
*/
type ListAnexiaTemplatesOK struct {
	Payload models.AnexiaTemplateList
}

func (o *ListAnexiaTemplatesOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/providers/anexia/templates][%d] listAnexiaTemplatesOK  %+v", 200, o.Payload)
}
func (o *ListAnexiaTemplatesOK) GetPayload() models.AnexiaTemplateList {
	return o.Payload
}

func (o *ListAnexiaTemplatesOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListAnexiaTemplatesDefault creates a ListAnexiaTemplatesDefault with default headers values
func NewListAnexiaTemplatesDefault(code int) *ListAnexiaTemplatesDefault {
	return &ListAnexiaTemplatesDefault{
		_statusCode: code,
	}
}

/* ListAnexiaTemplatesDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListAnexiaTemplatesDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list anexia templates default response
func (o *ListAnexiaTemplatesDefault) Code() int {
	return o._statusCode
}

func (o *ListAnexiaTemplatesDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/providers/anexia/templates][%d] listAnexiaTemplates default  %+v", o._statusCode, o.Payload)
}
func (o *ListAnexiaTemplatesDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListAnexiaTemplatesDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
