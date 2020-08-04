// Code generated by go-swagger; DO NOT EDIT.

package datacenter

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/api/utils/apiclient/models"
)

// ListDCForSeedReader is a Reader for the ListDCForSeed structure.
type ListDCForSeedReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListDCForSeedReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListDCForSeedOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListDCForSeedUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListDCForSeedForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListDCForSeedDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListDCForSeedOK creates a ListDCForSeedOK with default headers values
func NewListDCForSeedOK() *ListDCForSeedOK {
	return &ListDCForSeedOK{}
}

/*ListDCForSeedOK handles this case with default header values.

Datacenter
*/
type ListDCForSeedOK struct {
	Payload []*models.Datacenter
}

func (o *ListDCForSeedOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/seed/{seed_name}/dc][%d] listDCForSeedOK  %+v", 200, o.Payload)
}

func (o *ListDCForSeedOK) GetPayload() []*models.Datacenter {
	return o.Payload
}

func (o *ListDCForSeedOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListDCForSeedUnauthorized creates a ListDCForSeedUnauthorized with default headers values
func NewListDCForSeedUnauthorized() *ListDCForSeedUnauthorized {
	return &ListDCForSeedUnauthorized{}
}

/*ListDCForSeedUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type ListDCForSeedUnauthorized struct {
}

func (o *ListDCForSeedUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/seed/{seed_name}/dc][%d] listDCForSeedUnauthorized ", 401)
}

func (o *ListDCForSeedUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListDCForSeedForbidden creates a ListDCForSeedForbidden with default headers values
func NewListDCForSeedForbidden() *ListDCForSeedForbidden {
	return &ListDCForSeedForbidden{}
}

/*ListDCForSeedForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type ListDCForSeedForbidden struct {
}

func (o *ListDCForSeedForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/seed/{seed_name}/dc][%d] listDCForSeedForbidden ", 403)
}

func (o *ListDCForSeedForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListDCForSeedDefault creates a ListDCForSeedDefault with default headers values
func NewListDCForSeedDefault(code int) *ListDCForSeedDefault {
	return &ListDCForSeedDefault{
		_statusCode: code,
	}
}

/*ListDCForSeedDefault handles this case with default header values.

errorResponse
*/
type ListDCForSeedDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list d c for seed default response
func (o *ListDCForSeedDefault) Code() int {
	return o._statusCode
}

func (o *ListDCForSeedDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/seed/{seed_name}/dc][%d] listDCForSeed default  %+v", o._statusCode, o.Payload)
}

func (o *ListDCForSeedDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListDCForSeedDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
