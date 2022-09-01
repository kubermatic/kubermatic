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

// PatchAllowedRegistryReader is a Reader for the PatchAllowedRegistry structure.
type PatchAllowedRegistryReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *PatchAllowedRegistryReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewPatchAllowedRegistryOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewPatchAllowedRegistryUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewPatchAllowedRegistryForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewPatchAllowedRegistryDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewPatchAllowedRegistryOK creates a PatchAllowedRegistryOK with default headers values
func NewPatchAllowedRegistryOK() *PatchAllowedRegistryOK {
	return &PatchAllowedRegistryOK{}
}

/* PatchAllowedRegistryOK describes a response with status code 200, with default header values.

ConstraintTemplate
*/
type PatchAllowedRegistryOK struct {
	Payload *models.ConstraintTemplate
}

// IsSuccess returns true when this patch allowed registry o k response has a 2xx status code
func (o *PatchAllowedRegistryOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this patch allowed registry o k response has a 3xx status code
func (o *PatchAllowedRegistryOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this patch allowed registry o k response has a 4xx status code
func (o *PatchAllowedRegistryOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this patch allowed registry o k response has a 5xx status code
func (o *PatchAllowedRegistryOK) IsServerError() bool {
	return false
}

// IsCode returns true when this patch allowed registry o k response a status code equal to that given
func (o *PatchAllowedRegistryOK) IsCode(code int) bool {
	return code == 200
}

func (o *PatchAllowedRegistryOK) Error() string {
	return fmt.Sprintf("[PATCH /api/v2/allowedregistries/{allowed_registry}][%d] patchAllowedRegistryOK  %+v", 200, o.Payload)
}

func (o *PatchAllowedRegistryOK) String() string {
	return fmt.Sprintf("[PATCH /api/v2/allowedregistries/{allowed_registry}][%d] patchAllowedRegistryOK  %+v", 200, o.Payload)
}

func (o *PatchAllowedRegistryOK) GetPayload() *models.ConstraintTemplate {
	return o.Payload
}

func (o *PatchAllowedRegistryOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ConstraintTemplate)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPatchAllowedRegistryUnauthorized creates a PatchAllowedRegistryUnauthorized with default headers values
func NewPatchAllowedRegistryUnauthorized() *PatchAllowedRegistryUnauthorized {
	return &PatchAllowedRegistryUnauthorized{}
}

/* PatchAllowedRegistryUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type PatchAllowedRegistryUnauthorized struct {
}

// IsSuccess returns true when this patch allowed registry unauthorized response has a 2xx status code
func (o *PatchAllowedRegistryUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this patch allowed registry unauthorized response has a 3xx status code
func (o *PatchAllowedRegistryUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this patch allowed registry unauthorized response has a 4xx status code
func (o *PatchAllowedRegistryUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this patch allowed registry unauthorized response has a 5xx status code
func (o *PatchAllowedRegistryUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this patch allowed registry unauthorized response a status code equal to that given
func (o *PatchAllowedRegistryUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *PatchAllowedRegistryUnauthorized) Error() string {
	return fmt.Sprintf("[PATCH /api/v2/allowedregistries/{allowed_registry}][%d] patchAllowedRegistryUnauthorized ", 401)
}

func (o *PatchAllowedRegistryUnauthorized) String() string {
	return fmt.Sprintf("[PATCH /api/v2/allowedregistries/{allowed_registry}][%d] patchAllowedRegistryUnauthorized ", 401)
}

func (o *PatchAllowedRegistryUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewPatchAllowedRegistryForbidden creates a PatchAllowedRegistryForbidden with default headers values
func NewPatchAllowedRegistryForbidden() *PatchAllowedRegistryForbidden {
	return &PatchAllowedRegistryForbidden{}
}

/* PatchAllowedRegistryForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type PatchAllowedRegistryForbidden struct {
}

// IsSuccess returns true when this patch allowed registry forbidden response has a 2xx status code
func (o *PatchAllowedRegistryForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this patch allowed registry forbidden response has a 3xx status code
func (o *PatchAllowedRegistryForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this patch allowed registry forbidden response has a 4xx status code
func (o *PatchAllowedRegistryForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this patch allowed registry forbidden response has a 5xx status code
func (o *PatchAllowedRegistryForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this patch allowed registry forbidden response a status code equal to that given
func (o *PatchAllowedRegistryForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *PatchAllowedRegistryForbidden) Error() string {
	return fmt.Sprintf("[PATCH /api/v2/allowedregistries/{allowed_registry}][%d] patchAllowedRegistryForbidden ", 403)
}

func (o *PatchAllowedRegistryForbidden) String() string {
	return fmt.Sprintf("[PATCH /api/v2/allowedregistries/{allowed_registry}][%d] patchAllowedRegistryForbidden ", 403)
}

func (o *PatchAllowedRegistryForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewPatchAllowedRegistryDefault creates a PatchAllowedRegistryDefault with default headers values
func NewPatchAllowedRegistryDefault(code int) *PatchAllowedRegistryDefault {
	return &PatchAllowedRegistryDefault{
		_statusCode: code,
	}
}

/* PatchAllowedRegistryDefault describes a response with status code -1, with default header values.

errorResponse
*/
type PatchAllowedRegistryDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the patch allowed registry default response
func (o *PatchAllowedRegistryDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this patch allowed registry default response has a 2xx status code
func (o *PatchAllowedRegistryDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this patch allowed registry default response has a 3xx status code
func (o *PatchAllowedRegistryDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this patch allowed registry default response has a 4xx status code
func (o *PatchAllowedRegistryDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this patch allowed registry default response has a 5xx status code
func (o *PatchAllowedRegistryDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this patch allowed registry default response a status code equal to that given
func (o *PatchAllowedRegistryDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *PatchAllowedRegistryDefault) Error() string {
	return fmt.Sprintf("[PATCH /api/v2/allowedregistries/{allowed_registry}][%d] patchAllowedRegistry default  %+v", o._statusCode, o.Payload)
}

func (o *PatchAllowedRegistryDefault) String() string {
	return fmt.Sprintf("[PATCH /api/v2/allowedregistries/{allowed_registry}][%d] patchAllowedRegistry default  %+v", o._statusCode, o.Payload)
}

func (o *PatchAllowedRegistryDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *PatchAllowedRegistryDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
