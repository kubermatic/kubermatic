// Code generated by go-swagger; DO NOT EDIT.

package resource_quota

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// PatchResourceQuotaReader is a Reader for the PatchResourceQuota structure.
type PatchResourceQuotaReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *PatchResourceQuotaReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewPatchResourceQuotaOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewPatchResourceQuotaUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewPatchResourceQuotaForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewPatchResourceQuotaDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewPatchResourceQuotaOK creates a PatchResourceQuotaOK with default headers values
func NewPatchResourceQuotaOK() *PatchResourceQuotaOK {
	return &PatchResourceQuotaOK{}
}

/* PatchResourceQuotaOK describes a response with status code 200, with default header values.

EmptyResponse is a empty response
*/
type PatchResourceQuotaOK struct {
}

func (o *PatchResourceQuotaOK) Error() string {
	return fmt.Sprintf("[PATCH /api/v2/quotas/{quota_name}][%d] patchResourceQuotaOK ", 200)
}

func (o *PatchResourceQuotaOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewPatchResourceQuotaUnauthorized creates a PatchResourceQuotaUnauthorized with default headers values
func NewPatchResourceQuotaUnauthorized() *PatchResourceQuotaUnauthorized {
	return &PatchResourceQuotaUnauthorized{}
}

/* PatchResourceQuotaUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type PatchResourceQuotaUnauthorized struct {
}

func (o *PatchResourceQuotaUnauthorized) Error() string {
	return fmt.Sprintf("[PATCH /api/v2/quotas/{quota_name}][%d] patchResourceQuotaUnauthorized ", 401)
}

func (o *PatchResourceQuotaUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewPatchResourceQuotaForbidden creates a PatchResourceQuotaForbidden with default headers values
func NewPatchResourceQuotaForbidden() *PatchResourceQuotaForbidden {
	return &PatchResourceQuotaForbidden{}
}

/* PatchResourceQuotaForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type PatchResourceQuotaForbidden struct {
}

func (o *PatchResourceQuotaForbidden) Error() string {
	return fmt.Sprintf("[PATCH /api/v2/quotas/{quota_name}][%d] patchResourceQuotaForbidden ", 403)
}

func (o *PatchResourceQuotaForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewPatchResourceQuotaDefault creates a PatchResourceQuotaDefault with default headers values
func NewPatchResourceQuotaDefault(code int) *PatchResourceQuotaDefault {
	return &PatchResourceQuotaDefault{
		_statusCode: code,
	}
}

/* PatchResourceQuotaDefault describes a response with status code -1, with default header values.

errorResponse
*/
type PatchResourceQuotaDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the patch resource quota default response
func (o *PatchResourceQuotaDefault) Code() int {
	return o._statusCode
}

func (o *PatchResourceQuotaDefault) Error() string {
	return fmt.Sprintf("[PATCH /api/v2/quotas/{quota_name}][%d] patchResourceQuota default  %+v", o._statusCode, o.Payload)
}
func (o *PatchResourceQuotaDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *PatchResourceQuotaDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
