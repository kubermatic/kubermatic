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

// GetClusterReader is a Reader for the GetCluster structure.
type GetClusterReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetClusterReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetClusterOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewGetClusterUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewGetClusterForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewGetClusterDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetClusterOK creates a GetClusterOK with default headers values
func NewGetClusterOK() *GetClusterOK {
	return &GetClusterOK{}
}

/* GetClusterOK describes a response with status code 200, with default header values.

Cluster
*/
type GetClusterOK struct {
	Payload *models.Cluster
}

// IsSuccess returns true when this get cluster o k response has a 2xx status code
func (o *GetClusterOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this get cluster o k response has a 3xx status code
func (o *GetClusterOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get cluster o k response has a 4xx status code
func (o *GetClusterOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this get cluster o k response has a 5xx status code
func (o *GetClusterOK) IsServerError() bool {
	return false
}

// IsCode returns true when this get cluster o k response a status code equal to that given
func (o *GetClusterOK) IsCode(code int) bool {
	return code == 200
}

func (o *GetClusterOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}][%d] getClusterOK  %+v", 200, o.Payload)
}

func (o *GetClusterOK) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}][%d] getClusterOK  %+v", 200, o.Payload)
}

func (o *GetClusterOK) GetPayload() *models.Cluster {
	return o.Payload
}

func (o *GetClusterOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Cluster)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetClusterUnauthorized creates a GetClusterUnauthorized with default headers values
func NewGetClusterUnauthorized() *GetClusterUnauthorized {
	return &GetClusterUnauthorized{}
}

/* GetClusterUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type GetClusterUnauthorized struct {
}

// IsSuccess returns true when this get cluster unauthorized response has a 2xx status code
func (o *GetClusterUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this get cluster unauthorized response has a 3xx status code
func (o *GetClusterUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get cluster unauthorized response has a 4xx status code
func (o *GetClusterUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this get cluster unauthorized response has a 5xx status code
func (o *GetClusterUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this get cluster unauthorized response a status code equal to that given
func (o *GetClusterUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *GetClusterUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}][%d] getClusterUnauthorized ", 401)
}

func (o *GetClusterUnauthorized) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}][%d] getClusterUnauthorized ", 401)
}

func (o *GetClusterUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetClusterForbidden creates a GetClusterForbidden with default headers values
func NewGetClusterForbidden() *GetClusterForbidden {
	return &GetClusterForbidden{}
}

/* GetClusterForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type GetClusterForbidden struct {
}

// IsSuccess returns true when this get cluster forbidden response has a 2xx status code
func (o *GetClusterForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this get cluster forbidden response has a 3xx status code
func (o *GetClusterForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get cluster forbidden response has a 4xx status code
func (o *GetClusterForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this get cluster forbidden response has a 5xx status code
func (o *GetClusterForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this get cluster forbidden response a status code equal to that given
func (o *GetClusterForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *GetClusterForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}][%d] getClusterForbidden ", 403)
}

func (o *GetClusterForbidden) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}][%d] getClusterForbidden ", 403)
}

func (o *GetClusterForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetClusterDefault creates a GetClusterDefault with default headers values
func NewGetClusterDefault(code int) *GetClusterDefault {
	return &GetClusterDefault{
		_statusCode: code,
	}
}

/* GetClusterDefault describes a response with status code -1, with default header values.

errorResponse
*/
type GetClusterDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the get cluster default response
func (o *GetClusterDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this get cluster default response has a 2xx status code
func (o *GetClusterDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this get cluster default response has a 3xx status code
func (o *GetClusterDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this get cluster default response has a 4xx status code
func (o *GetClusterDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this get cluster default response has a 5xx status code
func (o *GetClusterDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this get cluster default response a status code equal to that given
func (o *GetClusterDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *GetClusterDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}][%d] getCluster default  %+v", o._statusCode, o.Payload)
}

func (o *GetClusterDefault) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}][%d] getCluster default  %+v", o._statusCode, o.Payload)
}

func (o *GetClusterDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *GetClusterDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
