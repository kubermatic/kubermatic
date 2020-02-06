// Code generated by go-swagger; DO NOT EDIT.

package serviceaccounts

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// DeleteServiceAccountReader is a Reader for the DeleteServiceAccount structure.
type DeleteServiceAccountReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeleteServiceAccountReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewDeleteServiceAccountOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewDeleteServiceAccountUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewDeleteServiceAccountForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewDeleteServiceAccountDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewDeleteServiceAccountOK creates a DeleteServiceAccountOK with default headers values
func NewDeleteServiceAccountOK() *DeleteServiceAccountOK {
	return &DeleteServiceAccountOK{}
}

/*DeleteServiceAccountOK handles this case with default header values.

EmptyResponse is a empty response
*/
type DeleteServiceAccountOK struct {
}

func (o *DeleteServiceAccountOK) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}][%d] deleteServiceAccountOK ", 200)
}

func (o *DeleteServiceAccountOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteServiceAccountUnauthorized creates a DeleteServiceAccountUnauthorized with default headers values
func NewDeleteServiceAccountUnauthorized() *DeleteServiceAccountUnauthorized {
	return &DeleteServiceAccountUnauthorized{}
}

/*DeleteServiceAccountUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type DeleteServiceAccountUnauthorized struct {
}

func (o *DeleteServiceAccountUnauthorized) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}][%d] deleteServiceAccountUnauthorized ", 401)
}

func (o *DeleteServiceAccountUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteServiceAccountForbidden creates a DeleteServiceAccountForbidden with default headers values
func NewDeleteServiceAccountForbidden() *DeleteServiceAccountForbidden {
	return &DeleteServiceAccountForbidden{}
}

/*DeleteServiceAccountForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type DeleteServiceAccountForbidden struct {
}

func (o *DeleteServiceAccountForbidden) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}][%d] deleteServiceAccountForbidden ", 403)
}

func (o *DeleteServiceAccountForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteServiceAccountDefault creates a DeleteServiceAccountDefault with default headers values
func NewDeleteServiceAccountDefault(code int) *DeleteServiceAccountDefault {
	return &DeleteServiceAccountDefault{
		_statusCode: code,
	}
}

/*DeleteServiceAccountDefault handles this case with default header values.

errorResponse
*/
type DeleteServiceAccountDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the delete service account default response
func (o *DeleteServiceAccountDefault) Code() int {
	return o._statusCode
}

func (o *DeleteServiceAccountDefault) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}][%d] deleteServiceAccount default  %+v", o._statusCode, o.Payload)
}

func (o *DeleteServiceAccountDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *DeleteServiceAccountDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
