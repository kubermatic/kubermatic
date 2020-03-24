// Code generated by go-swagger; DO NOT EDIT.

package project

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// UpdateProjectReader is a Reader for the UpdateProject structure.
type UpdateProjectReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *UpdateProjectReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewUpdateProjectOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 400:
		result := NewUpdateProjectBadRequest()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 404:
		result := NewUpdateProjectNotFound()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 500:
		result := NewUpdateProjectInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 501:
		result := NewUpdateProjectNotImplemented()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewUpdateProjectDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewUpdateProjectOK creates a UpdateProjectOK with default headers values
func NewUpdateProjectOK() *UpdateProjectOK {
	return &UpdateProjectOK{}
}

/*UpdateProjectOK handles this case with default header values.

Project
*/
type UpdateProjectOK struct {
	Payload *models.Project
}

func (o *UpdateProjectOK) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}][%d] updateProjectOK  %+v", 200, o.Payload)
}

func (o *UpdateProjectOK) GetPayload() *models.Project {
	return o.Payload
}

func (o *UpdateProjectOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Project)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewUpdateProjectBadRequest creates a UpdateProjectBadRequest with default headers values
func NewUpdateProjectBadRequest() *UpdateProjectBadRequest {
	return &UpdateProjectBadRequest{}
}

/*UpdateProjectBadRequest handles this case with default header values.

EmptyResponse is a empty response
*/
type UpdateProjectBadRequest struct {
}

func (o *UpdateProjectBadRequest) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}][%d] updateProjectBadRequest ", 400)
}

func (o *UpdateProjectBadRequest) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewUpdateProjectNotFound creates a UpdateProjectNotFound with default headers values
func NewUpdateProjectNotFound() *UpdateProjectNotFound {
	return &UpdateProjectNotFound{}
}

/*UpdateProjectNotFound handles this case with default header values.

EmptyResponse is a empty response
*/
type UpdateProjectNotFound struct {
}

func (o *UpdateProjectNotFound) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}][%d] updateProjectNotFound ", 404)
}

func (o *UpdateProjectNotFound) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewUpdateProjectInternalServerError creates a UpdateProjectInternalServerError with default headers values
func NewUpdateProjectInternalServerError() *UpdateProjectInternalServerError {
	return &UpdateProjectInternalServerError{}
}

/*UpdateProjectInternalServerError handles this case with default header values.

EmptyResponse is a empty response
*/
type UpdateProjectInternalServerError struct {
}

func (o *UpdateProjectInternalServerError) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}][%d] updateProjectInternalServerError ", 500)
}

func (o *UpdateProjectInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewUpdateProjectNotImplemented creates a UpdateProjectNotImplemented with default headers values
func NewUpdateProjectNotImplemented() *UpdateProjectNotImplemented {
	return &UpdateProjectNotImplemented{}
}

/*UpdateProjectNotImplemented handles this case with default header values.

EmptyResponse is a empty response
*/
type UpdateProjectNotImplemented struct {
}

func (o *UpdateProjectNotImplemented) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}][%d] updateProjectNotImplemented ", 501)
}

func (o *UpdateProjectNotImplemented) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewUpdateProjectDefault creates a UpdateProjectDefault with default headers values
func NewUpdateProjectDefault(code int) *UpdateProjectDefault {
	return &UpdateProjectDefault{
		_statusCode: code,
	}
}

/*UpdateProjectDefault handles this case with default header values.

errorResponse
*/
type UpdateProjectDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the update project default response
func (o *UpdateProjectDefault) Code() int {
	return o._statusCode
}

func (o *UpdateProjectDefault) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}][%d] updateProject default  %+v", o._statusCode, o.Payload)
}

func (o *UpdateProjectDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *UpdateProjectDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
