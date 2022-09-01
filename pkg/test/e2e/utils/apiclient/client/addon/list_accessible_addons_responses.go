// Code generated by go-swagger; DO NOT EDIT.

package addon

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListAccessibleAddonsReader is a Reader for the ListAccessibleAddons structure.
type ListAccessibleAddonsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListAccessibleAddonsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListAccessibleAddonsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListAccessibleAddonsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListAccessibleAddonsForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListAccessibleAddonsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListAccessibleAddonsOK creates a ListAccessibleAddonsOK with default headers values
func NewListAccessibleAddonsOK() *ListAccessibleAddonsOK {
	return &ListAccessibleAddonsOK{}
}

/* ListAccessibleAddonsOK describes a response with status code 200, with default header values.

AccessibleAddons
*/
type ListAccessibleAddonsOK struct {
	Payload models.AccessibleAddons
}

// IsSuccess returns true when this list accessible addons o k response has a 2xx status code
func (o *ListAccessibleAddonsOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list accessible addons o k response has a 3xx status code
func (o *ListAccessibleAddonsOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list accessible addons o k response has a 4xx status code
func (o *ListAccessibleAddonsOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list accessible addons o k response has a 5xx status code
func (o *ListAccessibleAddonsOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list accessible addons o k response a status code equal to that given
func (o *ListAccessibleAddonsOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListAccessibleAddonsOK) Error() string {
	return fmt.Sprintf("[POST /api/v1/addons][%d] listAccessibleAddonsOK  %+v", 200, o.Payload)
}

func (o *ListAccessibleAddonsOK) String() string {
	return fmt.Sprintf("[POST /api/v1/addons][%d] listAccessibleAddonsOK  %+v", 200, o.Payload)
}

func (o *ListAccessibleAddonsOK) GetPayload() models.AccessibleAddons {
	return o.Payload
}

func (o *ListAccessibleAddonsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListAccessibleAddonsUnauthorized creates a ListAccessibleAddonsUnauthorized with default headers values
func NewListAccessibleAddonsUnauthorized() *ListAccessibleAddonsUnauthorized {
	return &ListAccessibleAddonsUnauthorized{}
}

/* ListAccessibleAddonsUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type ListAccessibleAddonsUnauthorized struct {
}

// IsSuccess returns true when this list accessible addons unauthorized response has a 2xx status code
func (o *ListAccessibleAddonsUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list accessible addons unauthorized response has a 3xx status code
func (o *ListAccessibleAddonsUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list accessible addons unauthorized response has a 4xx status code
func (o *ListAccessibleAddonsUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this list accessible addons unauthorized response has a 5xx status code
func (o *ListAccessibleAddonsUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this list accessible addons unauthorized response a status code equal to that given
func (o *ListAccessibleAddonsUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *ListAccessibleAddonsUnauthorized) Error() string {
	return fmt.Sprintf("[POST /api/v1/addons][%d] listAccessibleAddonsUnauthorized ", 401)
}

func (o *ListAccessibleAddonsUnauthorized) String() string {
	return fmt.Sprintf("[POST /api/v1/addons][%d] listAccessibleAddonsUnauthorized ", 401)
}

func (o *ListAccessibleAddonsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListAccessibleAddonsForbidden creates a ListAccessibleAddonsForbidden with default headers values
func NewListAccessibleAddonsForbidden() *ListAccessibleAddonsForbidden {
	return &ListAccessibleAddonsForbidden{}
}

/* ListAccessibleAddonsForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type ListAccessibleAddonsForbidden struct {
}

// IsSuccess returns true when this list accessible addons forbidden response has a 2xx status code
func (o *ListAccessibleAddonsForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list accessible addons forbidden response has a 3xx status code
func (o *ListAccessibleAddonsForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list accessible addons forbidden response has a 4xx status code
func (o *ListAccessibleAddonsForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this list accessible addons forbidden response has a 5xx status code
func (o *ListAccessibleAddonsForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this list accessible addons forbidden response a status code equal to that given
func (o *ListAccessibleAddonsForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *ListAccessibleAddonsForbidden) Error() string {
	return fmt.Sprintf("[POST /api/v1/addons][%d] listAccessibleAddonsForbidden ", 403)
}

func (o *ListAccessibleAddonsForbidden) String() string {
	return fmt.Sprintf("[POST /api/v1/addons][%d] listAccessibleAddonsForbidden ", 403)
}

func (o *ListAccessibleAddonsForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListAccessibleAddonsDefault creates a ListAccessibleAddonsDefault with default headers values
func NewListAccessibleAddonsDefault(code int) *ListAccessibleAddonsDefault {
	return &ListAccessibleAddonsDefault{
		_statusCode: code,
	}
}

/* ListAccessibleAddonsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListAccessibleAddonsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list accessible addons default response
func (o *ListAccessibleAddonsDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list accessible addons default response has a 2xx status code
func (o *ListAccessibleAddonsDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list accessible addons default response has a 3xx status code
func (o *ListAccessibleAddonsDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list accessible addons default response has a 4xx status code
func (o *ListAccessibleAddonsDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list accessible addons default response has a 5xx status code
func (o *ListAccessibleAddonsDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list accessible addons default response a status code equal to that given
func (o *ListAccessibleAddonsDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListAccessibleAddonsDefault) Error() string {
	return fmt.Sprintf("[POST /api/v1/addons][%d] listAccessibleAddons default  %+v", o._statusCode, o.Payload)
}

func (o *ListAccessibleAddonsDefault) String() string {
	return fmt.Sprintf("[POST /api/v1/addons][%d] listAccessibleAddons default  %+v", o._statusCode, o.Payload)
}

func (o *ListAccessibleAddonsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListAccessibleAddonsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
