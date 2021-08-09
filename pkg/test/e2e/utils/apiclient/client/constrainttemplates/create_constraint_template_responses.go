// Code generated by go-swagger; DO NOT EDIT.

package constrainttemplates

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// CreateConstraintTemplateReader is a Reader for the CreateConstraintTemplate structure.
type CreateConstraintTemplateReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *CreateConstraintTemplateReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewCreateConstraintTemplateOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewCreateConstraintTemplateUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewCreateConstraintTemplateForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewCreateConstraintTemplateDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewCreateConstraintTemplateOK creates a CreateConstraintTemplateOK with default headers values
func NewCreateConstraintTemplateOK() *CreateConstraintTemplateOK {
	return &CreateConstraintTemplateOK{}
}

/* CreateConstraintTemplateOK describes a response with status code 200, with default header values.

ConstraintTemplate
*/
type CreateConstraintTemplateOK struct {
	Payload *models.ConstraintTemplate
}

func (o *CreateConstraintTemplateOK) Error() string {
	return fmt.Sprintf("[POST /api/v2/constrainttemplates][%d] createConstraintTemplateOK  %+v", 200, o.Payload)
}
func (o *CreateConstraintTemplateOK) GetPayload() *models.ConstraintTemplate {
	return o.Payload
}

func (o *CreateConstraintTemplateOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ConstraintTemplate)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewCreateConstraintTemplateUnauthorized creates a CreateConstraintTemplateUnauthorized with default headers values
func NewCreateConstraintTemplateUnauthorized() *CreateConstraintTemplateUnauthorized {
	return &CreateConstraintTemplateUnauthorized{}
}

/* CreateConstraintTemplateUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type CreateConstraintTemplateUnauthorized struct {
}

func (o *CreateConstraintTemplateUnauthorized) Error() string {
	return fmt.Sprintf("[POST /api/v2/constrainttemplates][%d] createConstraintTemplateUnauthorized ", 401)
}

func (o *CreateConstraintTemplateUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateConstraintTemplateForbidden creates a CreateConstraintTemplateForbidden with default headers values
func NewCreateConstraintTemplateForbidden() *CreateConstraintTemplateForbidden {
	return &CreateConstraintTemplateForbidden{}
}

/* CreateConstraintTemplateForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type CreateConstraintTemplateForbidden struct {
}

func (o *CreateConstraintTemplateForbidden) Error() string {
	return fmt.Sprintf("[POST /api/v2/constrainttemplates][%d] createConstraintTemplateForbidden ", 403)
}

func (o *CreateConstraintTemplateForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateConstraintTemplateDefault creates a CreateConstraintTemplateDefault with default headers values
func NewCreateConstraintTemplateDefault(code int) *CreateConstraintTemplateDefault {
	return &CreateConstraintTemplateDefault{
		_statusCode: code,
	}
}

/* CreateConstraintTemplateDefault describes a response with status code -1, with default header values.

errorResponse
*/
type CreateConstraintTemplateDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the create constraint template default response
func (o *CreateConstraintTemplateDefault) Code() int {
	return o._statusCode
}

func (o *CreateConstraintTemplateDefault) Error() string {
	return fmt.Sprintf("[POST /api/v2/constrainttemplates][%d] createConstraintTemplate default  %+v", o._statusCode, o.Payload)
}
func (o *CreateConstraintTemplateDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *CreateConstraintTemplateDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
