// Code generated by go-swagger; DO NOT EDIT.

package constrainttemplates

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/api/utils/apiclient/models"
)

// DeleteConstraintTemplateReader is a Reader for the DeleteConstraintTemplate structure.
type DeleteConstraintTemplateReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeleteConstraintTemplateReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewDeleteConstraintTemplateOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewDeleteConstraintTemplateUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewDeleteConstraintTemplateForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewDeleteConstraintTemplateDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewDeleteConstraintTemplateOK creates a DeleteConstraintTemplateOK with default headers values
func NewDeleteConstraintTemplateOK() *DeleteConstraintTemplateOK {
	return &DeleteConstraintTemplateOK{}
}

/*DeleteConstraintTemplateOK handles this case with default header values.

EmptyResponse is a empty response
*/
type DeleteConstraintTemplateOK struct {
}

func (o *DeleteConstraintTemplateOK) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/constrainttemplates/{ct_name}][%d] deleteConstraintTemplateOK ", 200)
}

func (o *DeleteConstraintTemplateOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteConstraintTemplateUnauthorized creates a DeleteConstraintTemplateUnauthorized with default headers values
func NewDeleteConstraintTemplateUnauthorized() *DeleteConstraintTemplateUnauthorized {
	return &DeleteConstraintTemplateUnauthorized{}
}

/*DeleteConstraintTemplateUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type DeleteConstraintTemplateUnauthorized struct {
}

func (o *DeleteConstraintTemplateUnauthorized) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/constrainttemplates/{ct_name}][%d] deleteConstraintTemplateUnauthorized ", 401)
}

func (o *DeleteConstraintTemplateUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteConstraintTemplateForbidden creates a DeleteConstraintTemplateForbidden with default headers values
func NewDeleteConstraintTemplateForbidden() *DeleteConstraintTemplateForbidden {
	return &DeleteConstraintTemplateForbidden{}
}

/*DeleteConstraintTemplateForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type DeleteConstraintTemplateForbidden struct {
}

func (o *DeleteConstraintTemplateForbidden) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/constrainttemplates/{ct_name}][%d] deleteConstraintTemplateForbidden ", 403)
}

func (o *DeleteConstraintTemplateForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteConstraintTemplateDefault creates a DeleteConstraintTemplateDefault with default headers values
func NewDeleteConstraintTemplateDefault(code int) *DeleteConstraintTemplateDefault {
	return &DeleteConstraintTemplateDefault{
		_statusCode: code,
	}
}

/*DeleteConstraintTemplateDefault handles this case with default header values.

errorResponse
*/
type DeleteConstraintTemplateDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the delete constraint template default response
func (o *DeleteConstraintTemplateDefault) Code() int {
	return o._statusCode
}

func (o *DeleteConstraintTemplateDefault) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/constrainttemplates/{ct_name}][%d] deleteConstraintTemplate default  %+v", o._statusCode, o.Payload)
}

func (o *DeleteConstraintTemplateDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *DeleteConstraintTemplateDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
