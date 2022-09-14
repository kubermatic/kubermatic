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

// ListSSHKeysAssignedToClusterReader is a Reader for the ListSSHKeysAssignedToCluster structure.
type ListSSHKeysAssignedToClusterReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListSSHKeysAssignedToClusterReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListSSHKeysAssignedToClusterOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListSSHKeysAssignedToClusterUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListSSHKeysAssignedToClusterForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListSSHKeysAssignedToClusterDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListSSHKeysAssignedToClusterOK creates a ListSSHKeysAssignedToClusterOK with default headers values
func NewListSSHKeysAssignedToClusterOK() *ListSSHKeysAssignedToClusterOK {
	return &ListSSHKeysAssignedToClusterOK{}
}

/*
ListSSHKeysAssignedToClusterOK describes a response with status code 200, with default header values.

SSHKey
*/
type ListSSHKeysAssignedToClusterOK struct {
	Payload []*models.SSHKey
}

// IsSuccess returns true when this list Ssh keys assigned to cluster o k response has a 2xx status code
func (o *ListSSHKeysAssignedToClusterOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list Ssh keys assigned to cluster o k response has a 3xx status code
func (o *ListSSHKeysAssignedToClusterOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list Ssh keys assigned to cluster o k response has a 4xx status code
func (o *ListSSHKeysAssignedToClusterOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list Ssh keys assigned to cluster o k response has a 5xx status code
func (o *ListSSHKeysAssignedToClusterOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list Ssh keys assigned to cluster o k response a status code equal to that given
func (o *ListSSHKeysAssignedToClusterOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListSSHKeysAssignedToClusterOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/sshkeys][%d] listSshKeysAssignedToClusterOK  %+v", 200, o.Payload)
}

func (o *ListSSHKeysAssignedToClusterOK) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/sshkeys][%d] listSshKeysAssignedToClusterOK  %+v", 200, o.Payload)
}

func (o *ListSSHKeysAssignedToClusterOK) GetPayload() []*models.SSHKey {
	return o.Payload
}

func (o *ListSSHKeysAssignedToClusterOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListSSHKeysAssignedToClusterUnauthorized creates a ListSSHKeysAssignedToClusterUnauthorized with default headers values
func NewListSSHKeysAssignedToClusterUnauthorized() *ListSSHKeysAssignedToClusterUnauthorized {
	return &ListSSHKeysAssignedToClusterUnauthorized{}
}

/*
ListSSHKeysAssignedToClusterUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type ListSSHKeysAssignedToClusterUnauthorized struct {
}

// IsSuccess returns true when this list Ssh keys assigned to cluster unauthorized response has a 2xx status code
func (o *ListSSHKeysAssignedToClusterUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list Ssh keys assigned to cluster unauthorized response has a 3xx status code
func (o *ListSSHKeysAssignedToClusterUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list Ssh keys assigned to cluster unauthorized response has a 4xx status code
func (o *ListSSHKeysAssignedToClusterUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this list Ssh keys assigned to cluster unauthorized response has a 5xx status code
func (o *ListSSHKeysAssignedToClusterUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this list Ssh keys assigned to cluster unauthorized response a status code equal to that given
func (o *ListSSHKeysAssignedToClusterUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *ListSSHKeysAssignedToClusterUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/sshkeys][%d] listSshKeysAssignedToClusterUnauthorized ", 401)
}

func (o *ListSSHKeysAssignedToClusterUnauthorized) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/sshkeys][%d] listSshKeysAssignedToClusterUnauthorized ", 401)
}

func (o *ListSSHKeysAssignedToClusterUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListSSHKeysAssignedToClusterForbidden creates a ListSSHKeysAssignedToClusterForbidden with default headers values
func NewListSSHKeysAssignedToClusterForbidden() *ListSSHKeysAssignedToClusterForbidden {
	return &ListSSHKeysAssignedToClusterForbidden{}
}

/*
ListSSHKeysAssignedToClusterForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type ListSSHKeysAssignedToClusterForbidden struct {
}

// IsSuccess returns true when this list Ssh keys assigned to cluster forbidden response has a 2xx status code
func (o *ListSSHKeysAssignedToClusterForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list Ssh keys assigned to cluster forbidden response has a 3xx status code
func (o *ListSSHKeysAssignedToClusterForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list Ssh keys assigned to cluster forbidden response has a 4xx status code
func (o *ListSSHKeysAssignedToClusterForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this list Ssh keys assigned to cluster forbidden response has a 5xx status code
func (o *ListSSHKeysAssignedToClusterForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this list Ssh keys assigned to cluster forbidden response a status code equal to that given
func (o *ListSSHKeysAssignedToClusterForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *ListSSHKeysAssignedToClusterForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/sshkeys][%d] listSshKeysAssignedToClusterForbidden ", 403)
}

func (o *ListSSHKeysAssignedToClusterForbidden) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/sshkeys][%d] listSshKeysAssignedToClusterForbidden ", 403)
}

func (o *ListSSHKeysAssignedToClusterForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListSSHKeysAssignedToClusterDefault creates a ListSSHKeysAssignedToClusterDefault with default headers values
func NewListSSHKeysAssignedToClusterDefault(code int) *ListSSHKeysAssignedToClusterDefault {
	return &ListSSHKeysAssignedToClusterDefault{
		_statusCode: code,
	}
}

/*
ListSSHKeysAssignedToClusterDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListSSHKeysAssignedToClusterDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list SSH keys assigned to cluster default response
func (o *ListSSHKeysAssignedToClusterDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list SSH keys assigned to cluster default response has a 2xx status code
func (o *ListSSHKeysAssignedToClusterDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list SSH keys assigned to cluster default response has a 3xx status code
func (o *ListSSHKeysAssignedToClusterDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list SSH keys assigned to cluster default response has a 4xx status code
func (o *ListSSHKeysAssignedToClusterDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list SSH keys assigned to cluster default response has a 5xx status code
func (o *ListSSHKeysAssignedToClusterDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list SSH keys assigned to cluster default response a status code equal to that given
func (o *ListSSHKeysAssignedToClusterDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListSSHKeysAssignedToClusterDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/sshkeys][%d] listSSHKeysAssignedToCluster default  %+v", o._statusCode, o.Payload)
}

func (o *ListSSHKeysAssignedToClusterDefault) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/sshkeys][%d] listSSHKeysAssignedToCluster default  %+v", o._statusCode, o.Payload)
}

func (o *ListSSHKeysAssignedToClusterDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListSSHKeysAssignedToClusterDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
