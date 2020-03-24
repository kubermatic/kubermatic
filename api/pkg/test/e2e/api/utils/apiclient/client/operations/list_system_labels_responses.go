// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// ListSystemLabelsReader is a Reader for the ListSystemLabels structure.
type ListSystemLabelsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListSystemLabelsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListSystemLabelsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListSystemLabelsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListSystemLabelsForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListSystemLabelsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListSystemLabelsOK creates a ListSystemLabelsOK with default headers values
func NewListSystemLabelsOK() *ListSystemLabelsOK {
	return &ListSystemLabelsOK{}
}

/*ListSystemLabelsOK handles this case with default header values.

ResourceLabelMap
*/
type ListSystemLabelsOK struct {
	Payload models.ResourceLabelMap
}

func (o *ListSystemLabelsOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/labels/system][%d] listSystemLabelsOK  %+v", 200, o.Payload)
}

func (o *ListSystemLabelsOK) GetPayload() models.ResourceLabelMap {
	return o.Payload
}

func (o *ListSystemLabelsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListSystemLabelsUnauthorized creates a ListSystemLabelsUnauthorized with default headers values
func NewListSystemLabelsUnauthorized() *ListSystemLabelsUnauthorized {
	return &ListSystemLabelsUnauthorized{}
}

/*ListSystemLabelsUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type ListSystemLabelsUnauthorized struct {
}

func (o *ListSystemLabelsUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/labels/system][%d] listSystemLabelsUnauthorized ", 401)
}

func (o *ListSystemLabelsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListSystemLabelsForbidden creates a ListSystemLabelsForbidden with default headers values
func NewListSystemLabelsForbidden() *ListSystemLabelsForbidden {
	return &ListSystemLabelsForbidden{}
}

/*ListSystemLabelsForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type ListSystemLabelsForbidden struct {
}

func (o *ListSystemLabelsForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/labels/system][%d] listSystemLabelsForbidden ", 403)
}

func (o *ListSystemLabelsForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListSystemLabelsDefault creates a ListSystemLabelsDefault with default headers values
func NewListSystemLabelsDefault(code int) *ListSystemLabelsDefault {
	return &ListSystemLabelsDefault{
		_statusCode: code,
	}
}

/*ListSystemLabelsDefault handles this case with default header values.

errorResponse
*/
type ListSystemLabelsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list system labels default response
func (o *ListSystemLabelsDefault) Code() int {
	return o._statusCode
}

func (o *ListSystemLabelsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/labels/system][%d] listSystemLabels default  %+v", o._statusCode, o.Payload)
}

func (o *ListSystemLabelsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListSystemLabelsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
