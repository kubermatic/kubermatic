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

// DeleteConstraintReader is a Reader for the DeleteConstraint structure.
type DeleteConstraintReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeleteConstraintReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewDeleteConstraintOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewDeleteConstraintUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewDeleteConstraintForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewDeleteConstraintDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewDeleteConstraintOK creates a DeleteConstraintOK with default headers values
func NewDeleteConstraintOK() *DeleteConstraintOK {
	return &DeleteConstraintOK{}
}

/* DeleteConstraintOK describes a response with status code 200, with default header values.

EmptyResponse is a empty response
*/
type DeleteConstraintOK struct {
}

// IsSuccess returns true when this delete constraint o k response has a 2xx status code
func (o *DeleteConstraintOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this delete constraint o k response has a 3xx status code
func (o *DeleteConstraintOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this delete constraint o k response has a 4xx status code
func (o *DeleteConstraintOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this delete constraint o k response has a 5xx status code
func (o *DeleteConstraintOK) IsServerError() bool {
	return false
}

// IsCode returns true when this delete constraint o k response a status code equal to that given
func (o *DeleteConstraintOK) IsCode(code int) bool {
	return code == 200
}

func (o *DeleteConstraintOK) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints/{constraint_name}][%d] deleteConstraintOK ", 200)
}

func (o *DeleteConstraintOK) String() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints/{constraint_name}][%d] deleteConstraintOK ", 200)
}

func (o *DeleteConstraintOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteConstraintUnauthorized creates a DeleteConstraintUnauthorized with default headers values
func NewDeleteConstraintUnauthorized() *DeleteConstraintUnauthorized {
	return &DeleteConstraintUnauthorized{}
}

/* DeleteConstraintUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type DeleteConstraintUnauthorized struct {
}

// IsSuccess returns true when this delete constraint unauthorized response has a 2xx status code
func (o *DeleteConstraintUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this delete constraint unauthorized response has a 3xx status code
func (o *DeleteConstraintUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this delete constraint unauthorized response has a 4xx status code
func (o *DeleteConstraintUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this delete constraint unauthorized response has a 5xx status code
func (o *DeleteConstraintUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this delete constraint unauthorized response a status code equal to that given
func (o *DeleteConstraintUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *DeleteConstraintUnauthorized) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints/{constraint_name}][%d] deleteConstraintUnauthorized ", 401)
}

func (o *DeleteConstraintUnauthorized) String() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints/{constraint_name}][%d] deleteConstraintUnauthorized ", 401)
}

func (o *DeleteConstraintUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteConstraintForbidden creates a DeleteConstraintForbidden with default headers values
func NewDeleteConstraintForbidden() *DeleteConstraintForbidden {
	return &DeleteConstraintForbidden{}
}

/* DeleteConstraintForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type DeleteConstraintForbidden struct {
}

// IsSuccess returns true when this delete constraint forbidden response has a 2xx status code
func (o *DeleteConstraintForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this delete constraint forbidden response has a 3xx status code
func (o *DeleteConstraintForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this delete constraint forbidden response has a 4xx status code
func (o *DeleteConstraintForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this delete constraint forbidden response has a 5xx status code
func (o *DeleteConstraintForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this delete constraint forbidden response a status code equal to that given
func (o *DeleteConstraintForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *DeleteConstraintForbidden) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints/{constraint_name}][%d] deleteConstraintForbidden ", 403)
}

func (o *DeleteConstraintForbidden) String() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints/{constraint_name}][%d] deleteConstraintForbidden ", 403)
}

func (o *DeleteConstraintForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteConstraintDefault creates a DeleteConstraintDefault with default headers values
func NewDeleteConstraintDefault(code int) *DeleteConstraintDefault {
	return &DeleteConstraintDefault{
		_statusCode: code,
	}
}

/* DeleteConstraintDefault describes a response with status code -1, with default header values.

errorResponse
*/
type DeleteConstraintDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the delete constraint default response
func (o *DeleteConstraintDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this delete constraint default response has a 2xx status code
func (o *DeleteConstraintDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this delete constraint default response has a 3xx status code
func (o *DeleteConstraintDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this delete constraint default response has a 4xx status code
func (o *DeleteConstraintDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this delete constraint default response has a 5xx status code
func (o *DeleteConstraintDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this delete constraint default response a status code equal to that given
func (o *DeleteConstraintDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *DeleteConstraintDefault) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints/{constraint_name}][%d] deleteConstraint default  %+v", o._statusCode, o.Payload)
}

func (o *DeleteConstraintDefault) String() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints/{constraint_name}][%d] deleteConstraint default  %+v", o._statusCode, o.Payload)
}

func (o *DeleteConstraintDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *DeleteConstraintDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
