// Code generated by go-swagger; DO NOT EDIT.

package allowedregistries

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// DeleteAllowedRegistryReader is a Reader for the DeleteAllowedRegistry structure.
type DeleteAllowedRegistryReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeleteAllowedRegistryReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewDeleteAllowedRegistryOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewDeleteAllowedRegistryUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewDeleteAllowedRegistryForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewDeleteAllowedRegistryDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewDeleteAllowedRegistryOK creates a DeleteAllowedRegistryOK with default headers values
func NewDeleteAllowedRegistryOK() *DeleteAllowedRegistryOK {
	return &DeleteAllowedRegistryOK{}
}

/* DeleteAllowedRegistryOK describes a response with status code 200, with default header values.

EmptyResponse is a empty response
*/
type DeleteAllowedRegistryOK struct {
}

// IsSuccess returns true when this delete allowed registry o k response has a 2xx status code
func (o *DeleteAllowedRegistryOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this delete allowed registry o k response has a 3xx status code
func (o *DeleteAllowedRegistryOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this delete allowed registry o k response has a 4xx status code
func (o *DeleteAllowedRegistryOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this delete allowed registry o k response has a 5xx status code
func (o *DeleteAllowedRegistryOK) IsServerError() bool {
	return false
}

// IsCode returns true when this delete allowed registry o k response a status code equal to that given
func (o *DeleteAllowedRegistryOK) IsCode(code int) bool {
	return code == 200
}

func (o *DeleteAllowedRegistryOK) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/allowedregistries/{allowed_registry}][%d] deleteAllowedRegistryOK ", 200)
}

func (o *DeleteAllowedRegistryOK) String() string {
	return fmt.Sprintf("[DELETE /api/v2/allowedregistries/{allowed_registry}][%d] deleteAllowedRegistryOK ", 200)
}

func (o *DeleteAllowedRegistryOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteAllowedRegistryUnauthorized creates a DeleteAllowedRegistryUnauthorized with default headers values
func NewDeleteAllowedRegistryUnauthorized() *DeleteAllowedRegistryUnauthorized {
	return &DeleteAllowedRegistryUnauthorized{}
}

/* DeleteAllowedRegistryUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type DeleteAllowedRegistryUnauthorized struct {
}

// IsSuccess returns true when this delete allowed registry unauthorized response has a 2xx status code
func (o *DeleteAllowedRegistryUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this delete allowed registry unauthorized response has a 3xx status code
func (o *DeleteAllowedRegistryUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this delete allowed registry unauthorized response has a 4xx status code
func (o *DeleteAllowedRegistryUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this delete allowed registry unauthorized response has a 5xx status code
func (o *DeleteAllowedRegistryUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this delete allowed registry unauthorized response a status code equal to that given
func (o *DeleteAllowedRegistryUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *DeleteAllowedRegistryUnauthorized) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/allowedregistries/{allowed_registry}][%d] deleteAllowedRegistryUnauthorized ", 401)
}

func (o *DeleteAllowedRegistryUnauthorized) String() string {
	return fmt.Sprintf("[DELETE /api/v2/allowedregistries/{allowed_registry}][%d] deleteAllowedRegistryUnauthorized ", 401)
}

func (o *DeleteAllowedRegistryUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteAllowedRegistryForbidden creates a DeleteAllowedRegistryForbidden with default headers values
func NewDeleteAllowedRegistryForbidden() *DeleteAllowedRegistryForbidden {
	return &DeleteAllowedRegistryForbidden{}
}

/* DeleteAllowedRegistryForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type DeleteAllowedRegistryForbidden struct {
}

// IsSuccess returns true when this delete allowed registry forbidden response has a 2xx status code
func (o *DeleteAllowedRegistryForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this delete allowed registry forbidden response has a 3xx status code
func (o *DeleteAllowedRegistryForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this delete allowed registry forbidden response has a 4xx status code
func (o *DeleteAllowedRegistryForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this delete allowed registry forbidden response has a 5xx status code
func (o *DeleteAllowedRegistryForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this delete allowed registry forbidden response a status code equal to that given
func (o *DeleteAllowedRegistryForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *DeleteAllowedRegistryForbidden) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/allowedregistries/{allowed_registry}][%d] deleteAllowedRegistryForbidden ", 403)
}

func (o *DeleteAllowedRegistryForbidden) String() string {
	return fmt.Sprintf("[DELETE /api/v2/allowedregistries/{allowed_registry}][%d] deleteAllowedRegistryForbidden ", 403)
}

func (o *DeleteAllowedRegistryForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteAllowedRegistryDefault creates a DeleteAllowedRegistryDefault with default headers values
func NewDeleteAllowedRegistryDefault(code int) *DeleteAllowedRegistryDefault {
	return &DeleteAllowedRegistryDefault{
		_statusCode: code,
	}
}

/* DeleteAllowedRegistryDefault describes a response with status code -1, with default header values.

errorResponse
*/
type DeleteAllowedRegistryDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the delete allowed registry default response
func (o *DeleteAllowedRegistryDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this delete allowed registry default response has a 2xx status code
func (o *DeleteAllowedRegistryDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this delete allowed registry default response has a 3xx status code
func (o *DeleteAllowedRegistryDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this delete allowed registry default response has a 4xx status code
func (o *DeleteAllowedRegistryDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this delete allowed registry default response has a 5xx status code
func (o *DeleteAllowedRegistryDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this delete allowed registry default response a status code equal to that given
func (o *DeleteAllowedRegistryDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *DeleteAllowedRegistryDefault) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/allowedregistries/{allowed_registry}][%d] deleteAllowedRegistry default  %+v", o._statusCode, o.Payload)
}

func (o *DeleteAllowedRegistryDefault) String() string {
	return fmt.Sprintf("[DELETE /api/v2/allowedregistries/{allowed_registry}][%d] deleteAllowedRegistry default  %+v", o._statusCode, o.Payload)
}

func (o *DeleteAllowedRegistryDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *DeleteAllowedRegistryDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
