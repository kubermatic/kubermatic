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

// ListCNIPluginVersionsForClusterReader is a Reader for the ListCNIPluginVersionsForCluster structure.
type ListCNIPluginVersionsForClusterReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListCNIPluginVersionsForClusterReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListCNIPluginVersionsForClusterOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListCNIPluginVersionsForClusterUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListCNIPluginVersionsForClusterForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListCNIPluginVersionsForClusterDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListCNIPluginVersionsForClusterOK creates a ListCNIPluginVersionsForClusterOK with default headers values
func NewListCNIPluginVersionsForClusterOK() *ListCNIPluginVersionsForClusterOK {
	return &ListCNIPluginVersionsForClusterOK{}
}

/* ListCNIPluginVersionsForClusterOK describes a response with status code 200, with default header values.

CNIVersions
*/
type ListCNIPluginVersionsForClusterOK struct {
	Payload *models.CNIVersions
}

// IsSuccess returns true when this list c n i plugin versions for cluster o k response has a 2xx status code
func (o *ListCNIPluginVersionsForClusterOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list c n i plugin versions for cluster o k response has a 3xx status code
func (o *ListCNIPluginVersionsForClusterOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list c n i plugin versions for cluster o k response has a 4xx status code
func (o *ListCNIPluginVersionsForClusterOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list c n i plugin versions for cluster o k response has a 5xx status code
func (o *ListCNIPluginVersionsForClusterOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list c n i plugin versions for cluster o k response a status code equal to that given
func (o *ListCNIPluginVersionsForClusterOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListCNIPluginVersionsForClusterOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/cniversions][%d] listCNIPluginVersionsForClusterOK  %+v", 200, o.Payload)
}

func (o *ListCNIPluginVersionsForClusterOK) String() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/cniversions][%d] listCNIPluginVersionsForClusterOK  %+v", 200, o.Payload)
}

func (o *ListCNIPluginVersionsForClusterOK) GetPayload() *models.CNIVersions {
	return o.Payload
}

func (o *ListCNIPluginVersionsForClusterOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.CNIVersions)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListCNIPluginVersionsForClusterUnauthorized creates a ListCNIPluginVersionsForClusterUnauthorized with default headers values
func NewListCNIPluginVersionsForClusterUnauthorized() *ListCNIPluginVersionsForClusterUnauthorized {
	return &ListCNIPluginVersionsForClusterUnauthorized{}
}

/* ListCNIPluginVersionsForClusterUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type ListCNIPluginVersionsForClusterUnauthorized struct {
}

// IsSuccess returns true when this list c n i plugin versions for cluster unauthorized response has a 2xx status code
func (o *ListCNIPluginVersionsForClusterUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list c n i plugin versions for cluster unauthorized response has a 3xx status code
func (o *ListCNIPluginVersionsForClusterUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list c n i plugin versions for cluster unauthorized response has a 4xx status code
func (o *ListCNIPluginVersionsForClusterUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this list c n i plugin versions for cluster unauthorized response has a 5xx status code
func (o *ListCNIPluginVersionsForClusterUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this list c n i plugin versions for cluster unauthorized response a status code equal to that given
func (o *ListCNIPluginVersionsForClusterUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *ListCNIPluginVersionsForClusterUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/cniversions][%d] listCNIPluginVersionsForClusterUnauthorized ", 401)
}

func (o *ListCNIPluginVersionsForClusterUnauthorized) String() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/cniversions][%d] listCNIPluginVersionsForClusterUnauthorized ", 401)
}

func (o *ListCNIPluginVersionsForClusterUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListCNIPluginVersionsForClusterForbidden creates a ListCNIPluginVersionsForClusterForbidden with default headers values
func NewListCNIPluginVersionsForClusterForbidden() *ListCNIPluginVersionsForClusterForbidden {
	return &ListCNIPluginVersionsForClusterForbidden{}
}

/* ListCNIPluginVersionsForClusterForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type ListCNIPluginVersionsForClusterForbidden struct {
}

// IsSuccess returns true when this list c n i plugin versions for cluster forbidden response has a 2xx status code
func (o *ListCNIPluginVersionsForClusterForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list c n i plugin versions for cluster forbidden response has a 3xx status code
func (o *ListCNIPluginVersionsForClusterForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list c n i plugin versions for cluster forbidden response has a 4xx status code
func (o *ListCNIPluginVersionsForClusterForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this list c n i plugin versions for cluster forbidden response has a 5xx status code
func (o *ListCNIPluginVersionsForClusterForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this list c n i plugin versions for cluster forbidden response a status code equal to that given
func (o *ListCNIPluginVersionsForClusterForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *ListCNIPluginVersionsForClusterForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/cniversions][%d] listCNIPluginVersionsForClusterForbidden ", 403)
}

func (o *ListCNIPluginVersionsForClusterForbidden) String() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/cniversions][%d] listCNIPluginVersionsForClusterForbidden ", 403)
}

func (o *ListCNIPluginVersionsForClusterForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListCNIPluginVersionsForClusterDefault creates a ListCNIPluginVersionsForClusterDefault with default headers values
func NewListCNIPluginVersionsForClusterDefault(code int) *ListCNIPluginVersionsForClusterDefault {
	return &ListCNIPluginVersionsForClusterDefault{
		_statusCode: code,
	}
}

/* ListCNIPluginVersionsForClusterDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListCNIPluginVersionsForClusterDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list c n i plugin versions for cluster default response
func (o *ListCNIPluginVersionsForClusterDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list c n i plugin versions for cluster default response has a 2xx status code
func (o *ListCNIPluginVersionsForClusterDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list c n i plugin versions for cluster default response has a 3xx status code
func (o *ListCNIPluginVersionsForClusterDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list c n i plugin versions for cluster default response has a 4xx status code
func (o *ListCNIPluginVersionsForClusterDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list c n i plugin versions for cluster default response has a 5xx status code
func (o *ListCNIPluginVersionsForClusterDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list c n i plugin versions for cluster default response a status code equal to that given
func (o *ListCNIPluginVersionsForClusterDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListCNIPluginVersionsForClusterDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/cniversions][%d] listCNIPluginVersionsForCluster default  %+v", o._statusCode, o.Payload)
}

func (o *ListCNIPluginVersionsForClusterDefault) String() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/cniversions][%d] listCNIPluginVersionsForCluster default  %+v", o._statusCode, o.Payload)
}

func (o *ListCNIPluginVersionsForClusterDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListCNIPluginVersionsForClusterDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
