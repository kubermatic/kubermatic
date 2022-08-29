// Code generated by go-swagger; DO NOT EDIT.

package project

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// DeleteClusterTemplateReader is a Reader for the DeleteClusterTemplate structure.
type DeleteClusterTemplateReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeleteClusterTemplateReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewDeleteClusterTemplateOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewDeleteClusterTemplateUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewDeleteClusterTemplateForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewDeleteClusterTemplateDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewDeleteClusterTemplateOK creates a DeleteClusterTemplateOK with default headers values
func NewDeleteClusterTemplateOK() *DeleteClusterTemplateOK {
	return &DeleteClusterTemplateOK{}
}

/*
DeleteClusterTemplateOK describes a response with status code 200, with default header values.

EmptyResponse is a empty response
*/
type DeleteClusterTemplateOK struct {
}

// IsSuccess returns true when this delete cluster template o k response has a 2xx status code
func (o *DeleteClusterTemplateOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this delete cluster template o k response has a 3xx status code
func (o *DeleteClusterTemplateOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this delete cluster template o k response has a 4xx status code
func (o *DeleteClusterTemplateOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this delete cluster template o k response has a 5xx status code
func (o *DeleteClusterTemplateOK) IsServerError() bool {
	return false
}

// IsCode returns true when this delete cluster template o k response a status code equal to that given
func (o *DeleteClusterTemplateOK) IsCode(code int) bool {
	return code == 200
}

func (o *DeleteClusterTemplateOK) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clustertemplates/{template_id}][%d] deleteClusterTemplateOK ", 200)
}

func (o *DeleteClusterTemplateOK) String() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clustertemplates/{template_id}][%d] deleteClusterTemplateOK ", 200)
}

func (o *DeleteClusterTemplateOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteClusterTemplateUnauthorized creates a DeleteClusterTemplateUnauthorized with default headers values
func NewDeleteClusterTemplateUnauthorized() *DeleteClusterTemplateUnauthorized {
	return &DeleteClusterTemplateUnauthorized{}
}

/*
DeleteClusterTemplateUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type DeleteClusterTemplateUnauthorized struct {
}

// IsSuccess returns true when this delete cluster template unauthorized response has a 2xx status code
func (o *DeleteClusterTemplateUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this delete cluster template unauthorized response has a 3xx status code
func (o *DeleteClusterTemplateUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this delete cluster template unauthorized response has a 4xx status code
func (o *DeleteClusterTemplateUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this delete cluster template unauthorized response has a 5xx status code
func (o *DeleteClusterTemplateUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this delete cluster template unauthorized response a status code equal to that given
func (o *DeleteClusterTemplateUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *DeleteClusterTemplateUnauthorized) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clustertemplates/{template_id}][%d] deleteClusterTemplateUnauthorized ", 401)
}

func (o *DeleteClusterTemplateUnauthorized) String() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clustertemplates/{template_id}][%d] deleteClusterTemplateUnauthorized ", 401)
}

func (o *DeleteClusterTemplateUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteClusterTemplateForbidden creates a DeleteClusterTemplateForbidden with default headers values
func NewDeleteClusterTemplateForbidden() *DeleteClusterTemplateForbidden {
	return &DeleteClusterTemplateForbidden{}
}

/*
DeleteClusterTemplateForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type DeleteClusterTemplateForbidden struct {
}

// IsSuccess returns true when this delete cluster template forbidden response has a 2xx status code
func (o *DeleteClusterTemplateForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this delete cluster template forbidden response has a 3xx status code
func (o *DeleteClusterTemplateForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this delete cluster template forbidden response has a 4xx status code
func (o *DeleteClusterTemplateForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this delete cluster template forbidden response has a 5xx status code
func (o *DeleteClusterTemplateForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this delete cluster template forbidden response a status code equal to that given
func (o *DeleteClusterTemplateForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *DeleteClusterTemplateForbidden) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clustertemplates/{template_id}][%d] deleteClusterTemplateForbidden ", 403)
}

func (o *DeleteClusterTemplateForbidden) String() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clustertemplates/{template_id}][%d] deleteClusterTemplateForbidden ", 403)
}

func (o *DeleteClusterTemplateForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteClusterTemplateDefault creates a DeleteClusterTemplateDefault with default headers values
func NewDeleteClusterTemplateDefault(code int) *DeleteClusterTemplateDefault {
	return &DeleteClusterTemplateDefault{
		_statusCode: code,
	}
}

/*
DeleteClusterTemplateDefault describes a response with status code -1, with default header values.

errorResponse
*/
type DeleteClusterTemplateDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the delete cluster template default response
func (o *DeleteClusterTemplateDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this delete cluster template default response has a 2xx status code
func (o *DeleteClusterTemplateDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this delete cluster template default response has a 3xx status code
func (o *DeleteClusterTemplateDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this delete cluster template default response has a 4xx status code
func (o *DeleteClusterTemplateDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this delete cluster template default response has a 5xx status code
func (o *DeleteClusterTemplateDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this delete cluster template default response a status code equal to that given
func (o *DeleteClusterTemplateDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *DeleteClusterTemplateDefault) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clustertemplates/{template_id}][%d] deleteClusterTemplate default  %+v", o._statusCode, o.Payload)
}

func (o *DeleteClusterTemplateDefault) String() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clustertemplates/{template_id}][%d] deleteClusterTemplate default  %+v", o._statusCode, o.Payload)
}

func (o *DeleteClusterTemplateDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *DeleteClusterTemplateDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
