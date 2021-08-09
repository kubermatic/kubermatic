// Code generated by go-swagger; DO NOT EDIT.

package admin

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// GetAdminsReader is a Reader for the GetAdmins structure.
type GetAdminsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetAdminsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetAdminsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewGetAdminsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewGetAdminsForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewGetAdminsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetAdminsOK creates a GetAdminsOK with default headers values
func NewGetAdminsOK() *GetAdminsOK {
	return &GetAdminsOK{}
}

/* GetAdminsOK describes a response with status code 200, with default header values.

Admin
*/
type GetAdminsOK struct {
	Payload []*models.Admin
}

func (o *GetAdminsOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin][%d] getAdminsOK  %+v", 200, o.Payload)
}
func (o *GetAdminsOK) GetPayload() []*models.Admin {
	return o.Payload
}

func (o *GetAdminsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetAdminsUnauthorized creates a GetAdminsUnauthorized with default headers values
func NewGetAdminsUnauthorized() *GetAdminsUnauthorized {
	return &GetAdminsUnauthorized{}
}

/* GetAdminsUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type GetAdminsUnauthorized struct {
}

func (o *GetAdminsUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin][%d] getAdminsUnauthorized ", 401)
}

func (o *GetAdminsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetAdminsForbidden creates a GetAdminsForbidden with default headers values
func NewGetAdminsForbidden() *GetAdminsForbidden {
	return &GetAdminsForbidden{}
}

/* GetAdminsForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type GetAdminsForbidden struct {
}

func (o *GetAdminsForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin][%d] getAdminsForbidden ", 403)
}

func (o *GetAdminsForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetAdminsDefault creates a GetAdminsDefault with default headers values
func NewGetAdminsDefault(code int) *GetAdminsDefault {
	return &GetAdminsDefault{
		_statusCode: code,
	}
}

/* GetAdminsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type GetAdminsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the get admins default response
func (o *GetAdminsDefault) Code() int {
	return o._statusCode
}

func (o *GetAdminsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin][%d] getAdmins default  %+v", o._statusCode, o.Payload)
}
func (o *GetAdminsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *GetAdminsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
