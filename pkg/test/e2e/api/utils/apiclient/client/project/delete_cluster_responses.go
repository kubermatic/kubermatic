// Code generated by go-swagger; DO NOT EDIT.

package project

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/api/utils/apiclient/models"
)

// DeleteClusterReader is a Reader for the DeleteCluster structure.
type DeleteClusterReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeleteClusterReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewDeleteClusterOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewDeleteClusterUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewDeleteClusterForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewDeleteClusterDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewDeleteClusterOK creates a DeleteClusterOK with default headers values
func NewDeleteClusterOK() *DeleteClusterOK {
	return &DeleteClusterOK{}
}

/*DeleteClusterOK handles this case with default header values.

EmptyResponse is a empty response
*/
type DeleteClusterOK struct {
}

func (o *DeleteClusterOK) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}][%d] deleteClusterOK ", 200)
}

func (o *DeleteClusterOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteClusterUnauthorized creates a DeleteClusterUnauthorized with default headers values
func NewDeleteClusterUnauthorized() *DeleteClusterUnauthorized {
	return &DeleteClusterUnauthorized{}
}

/*DeleteClusterUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type DeleteClusterUnauthorized struct {
}

func (o *DeleteClusterUnauthorized) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}][%d] deleteClusterUnauthorized ", 401)
}

func (o *DeleteClusterUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteClusterForbidden creates a DeleteClusterForbidden with default headers values
func NewDeleteClusterForbidden() *DeleteClusterForbidden {
	return &DeleteClusterForbidden{}
}

/*DeleteClusterForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type DeleteClusterForbidden struct {
}

func (o *DeleteClusterForbidden) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}][%d] deleteClusterForbidden ", 403)
}

func (o *DeleteClusterForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteClusterDefault creates a DeleteClusterDefault with default headers values
func NewDeleteClusterDefault(code int) *DeleteClusterDefault {
	return &DeleteClusterDefault{
		_statusCode: code,
	}
}

/*DeleteClusterDefault handles this case with default header values.

errorResponse
*/
type DeleteClusterDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the delete cluster default response
func (o *DeleteClusterDefault) Code() int {
	return o._statusCode
}

func (o *DeleteClusterDefault) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}][%d] deleteCluster default  %+v", o._statusCode, o.Payload)
}

func (o *DeleteClusterDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *DeleteClusterDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
