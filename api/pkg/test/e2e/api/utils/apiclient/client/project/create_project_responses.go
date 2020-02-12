// Code generated by go-swagger; DO NOT EDIT.

package project

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/swag"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// CreateProjectReader is a Reader for the CreateProject structure.
type CreateProjectReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *CreateProjectReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 201:
		result := NewCreateProjectCreated()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewCreateProjectUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 409:
		result := NewCreateProjectConflict()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewCreateProjectDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewCreateProjectCreated creates a CreateProjectCreated with default headers values
func NewCreateProjectCreated() *CreateProjectCreated {
	return &CreateProjectCreated{}
}

/*CreateProjectCreated handles this case with default header values.

Project
*/
type CreateProjectCreated struct {
	Payload *models.Project
}

func (o *CreateProjectCreated) Error() string {
	return fmt.Sprintf("[POST /api/v1/projects][%d] createProjectCreated  %+v", 201, o.Payload)
}

func (o *CreateProjectCreated) GetPayload() *models.Project {
	return o.Payload
}

func (o *CreateProjectCreated) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Project)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewCreateProjectUnauthorized creates a CreateProjectUnauthorized with default headers values
func NewCreateProjectUnauthorized() *CreateProjectUnauthorized {
	return &CreateProjectUnauthorized{}
}

/*CreateProjectUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type CreateProjectUnauthorized struct {
}

func (o *CreateProjectUnauthorized) Error() string {
	return fmt.Sprintf("[POST /api/v1/projects][%d] createProjectUnauthorized ", 401)
}

func (o *CreateProjectUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateProjectConflict creates a CreateProjectConflict with default headers values
func NewCreateProjectConflict() *CreateProjectConflict {
	return &CreateProjectConflict{}
}

/*CreateProjectConflict handles this case with default header values.

EmptyResponse is a empty response
*/
type CreateProjectConflict struct {
}

func (o *CreateProjectConflict) Error() string {
	return fmt.Sprintf("[POST /api/v1/projects][%d] createProjectConflict ", 409)
}

func (o *CreateProjectConflict) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateProjectDefault creates a CreateProjectDefault with default headers values
func NewCreateProjectDefault(code int) *CreateProjectDefault {
	return &CreateProjectDefault{
		_statusCode: code,
	}
}

/*CreateProjectDefault handles this case with default header values.

errorResponse
*/
type CreateProjectDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the create project default response
func (o *CreateProjectDefault) Code() int {
	return o._statusCode
}

func (o *CreateProjectDefault) Error() string {
	return fmt.Sprintf("[POST /api/v1/projects][%d] createProject default  %+v", o._statusCode, o.Payload)
}

func (o *CreateProjectDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *CreateProjectDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

/*CreateProjectBody create project body
swagger:model CreateProjectBody
*/
type CreateProjectBody struct {

	// labels
	Labels map[string]string `json:"labels,omitempty"`

	// name
	Name string `json:"name,omitempty"`
}

// Validate validates this create project body
func (o *CreateProjectBody) Validate(formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *CreateProjectBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *CreateProjectBody) UnmarshalBinary(b []byte) error {
	var res CreateProjectBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}
