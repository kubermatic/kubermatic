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

// GetConstraintTemplateReader is a Reader for the GetConstraintTemplate structure.
type GetConstraintTemplateReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetConstraintTemplateReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetConstraintTemplateOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewGetConstraintTemplateUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewGetConstraintTemplateForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewGetConstraintTemplateDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetConstraintTemplateOK creates a GetConstraintTemplateOK with default headers values
func NewGetConstraintTemplateOK() *GetConstraintTemplateOK {
	return &GetConstraintTemplateOK{}
}

/*
GetConstraintTemplateOK describes a response with status code 200, with default header values.

ConstraintTemplate
*/
type GetConstraintTemplateOK struct {
	Payload *models.ConstraintTemplate
}

// IsSuccess returns true when this get constraint template o k response has a 2xx status code
func (o *GetConstraintTemplateOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this get constraint template o k response has a 3xx status code
func (o *GetConstraintTemplateOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get constraint template o k response has a 4xx status code
func (o *GetConstraintTemplateOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this get constraint template o k response has a 5xx status code
func (o *GetConstraintTemplateOK) IsServerError() bool {
	return false
}

// IsCode returns true when this get constraint template o k response a status code equal to that given
func (o *GetConstraintTemplateOK) IsCode(code int) bool {
	return code == 200
}

func (o *GetConstraintTemplateOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/constrainttemplates/{ct_name}][%d] getConstraintTemplateOK  %+v", 200, o.Payload)
}

func (o *GetConstraintTemplateOK) String() string {
	return fmt.Sprintf("[GET /api/v2/constrainttemplates/{ct_name}][%d] getConstraintTemplateOK  %+v", 200, o.Payload)
}

func (o *GetConstraintTemplateOK) GetPayload() *models.ConstraintTemplate {
	return o.Payload
}

func (o *GetConstraintTemplateOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ConstraintTemplate)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetConstraintTemplateUnauthorized creates a GetConstraintTemplateUnauthorized with default headers values
func NewGetConstraintTemplateUnauthorized() *GetConstraintTemplateUnauthorized {
	return &GetConstraintTemplateUnauthorized{}
}

/*
GetConstraintTemplateUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type GetConstraintTemplateUnauthorized struct {
}

// IsSuccess returns true when this get constraint template unauthorized response has a 2xx status code
func (o *GetConstraintTemplateUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this get constraint template unauthorized response has a 3xx status code
func (o *GetConstraintTemplateUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get constraint template unauthorized response has a 4xx status code
func (o *GetConstraintTemplateUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this get constraint template unauthorized response has a 5xx status code
func (o *GetConstraintTemplateUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this get constraint template unauthorized response a status code equal to that given
func (o *GetConstraintTemplateUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *GetConstraintTemplateUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v2/constrainttemplates/{ct_name}][%d] getConstraintTemplateUnauthorized ", 401)
}

func (o *GetConstraintTemplateUnauthorized) String() string {
	return fmt.Sprintf("[GET /api/v2/constrainttemplates/{ct_name}][%d] getConstraintTemplateUnauthorized ", 401)
}

func (o *GetConstraintTemplateUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetConstraintTemplateForbidden creates a GetConstraintTemplateForbidden with default headers values
func NewGetConstraintTemplateForbidden() *GetConstraintTemplateForbidden {
	return &GetConstraintTemplateForbidden{}
}

/*
GetConstraintTemplateForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type GetConstraintTemplateForbidden struct {
}

// IsSuccess returns true when this get constraint template forbidden response has a 2xx status code
func (o *GetConstraintTemplateForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this get constraint template forbidden response has a 3xx status code
func (o *GetConstraintTemplateForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get constraint template forbidden response has a 4xx status code
func (o *GetConstraintTemplateForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this get constraint template forbidden response has a 5xx status code
func (o *GetConstraintTemplateForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this get constraint template forbidden response a status code equal to that given
func (o *GetConstraintTemplateForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *GetConstraintTemplateForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v2/constrainttemplates/{ct_name}][%d] getConstraintTemplateForbidden ", 403)
}

func (o *GetConstraintTemplateForbidden) String() string {
	return fmt.Sprintf("[GET /api/v2/constrainttemplates/{ct_name}][%d] getConstraintTemplateForbidden ", 403)
}

func (o *GetConstraintTemplateForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetConstraintTemplateDefault creates a GetConstraintTemplateDefault with default headers values
func NewGetConstraintTemplateDefault(code int) *GetConstraintTemplateDefault {
	return &GetConstraintTemplateDefault{
		_statusCode: code,
	}
}

/*
GetConstraintTemplateDefault describes a response with status code -1, with default header values.

errorResponse
*/
type GetConstraintTemplateDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the get constraint template default response
func (o *GetConstraintTemplateDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this get constraint template default response has a 2xx status code
func (o *GetConstraintTemplateDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this get constraint template default response has a 3xx status code
func (o *GetConstraintTemplateDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this get constraint template default response has a 4xx status code
func (o *GetConstraintTemplateDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this get constraint template default response has a 5xx status code
func (o *GetConstraintTemplateDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this get constraint template default response a status code equal to that given
func (o *GetConstraintTemplateDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *GetConstraintTemplateDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/constrainttemplates/{ct_name}][%d] getConstraintTemplate default  %+v", o._statusCode, o.Payload)
}

func (o *GetConstraintTemplateDefault) String() string {
	return fmt.Sprintf("[GET /api/v2/constrainttemplates/{ct_name}][%d] getConstraintTemplate default  %+v", o._statusCode, o.Payload)
}

func (o *GetConstraintTemplateDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *GetConstraintTemplateDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
