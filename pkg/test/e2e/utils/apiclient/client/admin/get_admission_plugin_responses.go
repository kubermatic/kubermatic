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

// GetAdmissionPluginReader is a Reader for the GetAdmissionPlugin structure.
type GetAdmissionPluginReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetAdmissionPluginReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetAdmissionPluginOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewGetAdmissionPluginUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewGetAdmissionPluginForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewGetAdmissionPluginDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetAdmissionPluginOK creates a GetAdmissionPluginOK with default headers values
func NewGetAdmissionPluginOK() *GetAdmissionPluginOK {
	return &GetAdmissionPluginOK{}
}

/* GetAdmissionPluginOK describes a response with status code 200, with default header values.

AdmissionPlugin
*/
type GetAdmissionPluginOK struct {
	Payload *models.AdmissionPlugin
}

// IsSuccess returns true when this get admission plugin o k response has a 2xx status code
func (o *GetAdmissionPluginOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this get admission plugin o k response has a 3xx status code
func (o *GetAdmissionPluginOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get admission plugin o k response has a 4xx status code
func (o *GetAdmissionPluginOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this get admission plugin o k response has a 5xx status code
func (o *GetAdmissionPluginOK) IsServerError() bool {
	return false
}

// IsCode returns true when this get admission plugin o k response a status code equal to that given
func (o *GetAdmissionPluginOK) IsCode(code int) bool {
	return code == 200
}

func (o *GetAdmissionPluginOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin/admission/plugins/{name}][%d] getAdmissionPluginOK  %+v", 200, o.Payload)
}

func (o *GetAdmissionPluginOK) String() string {
	return fmt.Sprintf("[GET /api/v1/admin/admission/plugins/{name}][%d] getAdmissionPluginOK  %+v", 200, o.Payload)
}

func (o *GetAdmissionPluginOK) GetPayload() *models.AdmissionPlugin {
	return o.Payload
}

func (o *GetAdmissionPluginOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.AdmissionPlugin)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetAdmissionPluginUnauthorized creates a GetAdmissionPluginUnauthorized with default headers values
func NewGetAdmissionPluginUnauthorized() *GetAdmissionPluginUnauthorized {
	return &GetAdmissionPluginUnauthorized{}
}

/* GetAdmissionPluginUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type GetAdmissionPluginUnauthorized struct {
}

// IsSuccess returns true when this get admission plugin unauthorized response has a 2xx status code
func (o *GetAdmissionPluginUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this get admission plugin unauthorized response has a 3xx status code
func (o *GetAdmissionPluginUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get admission plugin unauthorized response has a 4xx status code
func (o *GetAdmissionPluginUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this get admission plugin unauthorized response has a 5xx status code
func (o *GetAdmissionPluginUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this get admission plugin unauthorized response a status code equal to that given
func (o *GetAdmissionPluginUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *GetAdmissionPluginUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin/admission/plugins/{name}][%d] getAdmissionPluginUnauthorized ", 401)
}

func (o *GetAdmissionPluginUnauthorized) String() string {
	return fmt.Sprintf("[GET /api/v1/admin/admission/plugins/{name}][%d] getAdmissionPluginUnauthorized ", 401)
}

func (o *GetAdmissionPluginUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetAdmissionPluginForbidden creates a GetAdmissionPluginForbidden with default headers values
func NewGetAdmissionPluginForbidden() *GetAdmissionPluginForbidden {
	return &GetAdmissionPluginForbidden{}
}

/* GetAdmissionPluginForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type GetAdmissionPluginForbidden struct {
}

// IsSuccess returns true when this get admission plugin forbidden response has a 2xx status code
func (o *GetAdmissionPluginForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this get admission plugin forbidden response has a 3xx status code
func (o *GetAdmissionPluginForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get admission plugin forbidden response has a 4xx status code
func (o *GetAdmissionPluginForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this get admission plugin forbidden response has a 5xx status code
func (o *GetAdmissionPluginForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this get admission plugin forbidden response a status code equal to that given
func (o *GetAdmissionPluginForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *GetAdmissionPluginForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin/admission/plugins/{name}][%d] getAdmissionPluginForbidden ", 403)
}

func (o *GetAdmissionPluginForbidden) String() string {
	return fmt.Sprintf("[GET /api/v1/admin/admission/plugins/{name}][%d] getAdmissionPluginForbidden ", 403)
}

func (o *GetAdmissionPluginForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetAdmissionPluginDefault creates a GetAdmissionPluginDefault with default headers values
func NewGetAdmissionPluginDefault(code int) *GetAdmissionPluginDefault {
	return &GetAdmissionPluginDefault{
		_statusCode: code,
	}
}

/* GetAdmissionPluginDefault describes a response with status code -1, with default header values.

errorResponse
*/
type GetAdmissionPluginDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the get admission plugin default response
func (o *GetAdmissionPluginDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this get admission plugin default response has a 2xx status code
func (o *GetAdmissionPluginDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this get admission plugin default response has a 3xx status code
func (o *GetAdmissionPluginDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this get admission plugin default response has a 4xx status code
func (o *GetAdmissionPluginDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this get admission plugin default response has a 5xx status code
func (o *GetAdmissionPluginDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this get admission plugin default response a status code equal to that given
func (o *GetAdmissionPluginDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *GetAdmissionPluginDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin/admission/plugins/{name}][%d] getAdmissionPlugin default  %+v", o._statusCode, o.Payload)
}

func (o *GetAdmissionPluginDefault) String() string {
	return fmt.Sprintf("[GET /api/v1/admin/admission/plugins/{name}][%d] getAdmissionPlugin default  %+v", o._statusCode, o.Payload)
}

func (o *GetAdmissionPluginDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *GetAdmissionPluginDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
