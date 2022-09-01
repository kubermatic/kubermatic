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

// DeleteExternalClusterReader is a Reader for the DeleteExternalCluster structure.
type DeleteExternalClusterReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeleteExternalClusterReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewDeleteExternalClusterOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewDeleteExternalClusterUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewDeleteExternalClusterForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewDeleteExternalClusterDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewDeleteExternalClusterOK creates a DeleteExternalClusterOK with default headers values
func NewDeleteExternalClusterOK() *DeleteExternalClusterOK {
	return &DeleteExternalClusterOK{}
}

/* DeleteExternalClusterOK describes a response with status code 200, with default header values.

EmptyResponse is a empty response
*/
type DeleteExternalClusterOK struct {
}

// IsSuccess returns true when this delete external cluster o k response has a 2xx status code
func (o *DeleteExternalClusterOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this delete external cluster o k response has a 3xx status code
func (o *DeleteExternalClusterOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this delete external cluster o k response has a 4xx status code
func (o *DeleteExternalClusterOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this delete external cluster o k response has a 5xx status code
func (o *DeleteExternalClusterOK) IsServerError() bool {
	return false
}

// IsCode returns true when this delete external cluster o k response a status code equal to that given
func (o *DeleteExternalClusterOK) IsCode(code int) bool {
	return code == 200
}

func (o *DeleteExternalClusterOK) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}][%d] deleteExternalClusterOK ", 200)
}

func (o *DeleteExternalClusterOK) String() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}][%d] deleteExternalClusterOK ", 200)
}

func (o *DeleteExternalClusterOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteExternalClusterUnauthorized creates a DeleteExternalClusterUnauthorized with default headers values
func NewDeleteExternalClusterUnauthorized() *DeleteExternalClusterUnauthorized {
	return &DeleteExternalClusterUnauthorized{}
}

/* DeleteExternalClusterUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type DeleteExternalClusterUnauthorized struct {
}

// IsSuccess returns true when this delete external cluster unauthorized response has a 2xx status code
func (o *DeleteExternalClusterUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this delete external cluster unauthorized response has a 3xx status code
func (o *DeleteExternalClusterUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this delete external cluster unauthorized response has a 4xx status code
func (o *DeleteExternalClusterUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this delete external cluster unauthorized response has a 5xx status code
func (o *DeleteExternalClusterUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this delete external cluster unauthorized response a status code equal to that given
func (o *DeleteExternalClusterUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *DeleteExternalClusterUnauthorized) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}][%d] deleteExternalClusterUnauthorized ", 401)
}

func (o *DeleteExternalClusterUnauthorized) String() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}][%d] deleteExternalClusterUnauthorized ", 401)
}

func (o *DeleteExternalClusterUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteExternalClusterForbidden creates a DeleteExternalClusterForbidden with default headers values
func NewDeleteExternalClusterForbidden() *DeleteExternalClusterForbidden {
	return &DeleteExternalClusterForbidden{}
}

/* DeleteExternalClusterForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type DeleteExternalClusterForbidden struct {
}

// IsSuccess returns true when this delete external cluster forbidden response has a 2xx status code
func (o *DeleteExternalClusterForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this delete external cluster forbidden response has a 3xx status code
func (o *DeleteExternalClusterForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this delete external cluster forbidden response has a 4xx status code
func (o *DeleteExternalClusterForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this delete external cluster forbidden response has a 5xx status code
func (o *DeleteExternalClusterForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this delete external cluster forbidden response a status code equal to that given
func (o *DeleteExternalClusterForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *DeleteExternalClusterForbidden) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}][%d] deleteExternalClusterForbidden ", 403)
}

func (o *DeleteExternalClusterForbidden) String() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}][%d] deleteExternalClusterForbidden ", 403)
}

func (o *DeleteExternalClusterForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteExternalClusterDefault creates a DeleteExternalClusterDefault with default headers values
func NewDeleteExternalClusterDefault(code int) *DeleteExternalClusterDefault {
	return &DeleteExternalClusterDefault{
		_statusCode: code,
	}
}

/* DeleteExternalClusterDefault describes a response with status code -1, with default header values.

errorResponse
*/
type DeleteExternalClusterDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the delete external cluster default response
func (o *DeleteExternalClusterDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this delete external cluster default response has a 2xx status code
func (o *DeleteExternalClusterDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this delete external cluster default response has a 3xx status code
func (o *DeleteExternalClusterDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this delete external cluster default response has a 4xx status code
func (o *DeleteExternalClusterDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this delete external cluster default response has a 5xx status code
func (o *DeleteExternalClusterDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this delete external cluster default response a status code equal to that given
func (o *DeleteExternalClusterDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *DeleteExternalClusterDefault) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}][%d] deleteExternalCluster default  %+v", o._statusCode, o.Payload)
}

func (o *DeleteExternalClusterDefault) String() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}][%d] deleteExternalCluster default  %+v", o._statusCode, o.Payload)
}

func (o *DeleteExternalClusterDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *DeleteExternalClusterDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
