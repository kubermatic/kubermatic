// Code generated by go-swagger; DO NOT EDIT.

package project

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
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

/*ListSSHKeysAssignedToClusterOK handles this case with default header values.

SSHKey
*/
type ListSSHKeysAssignedToClusterOK struct {
	Payload []*models.SSHKey
}

func (o *ListSSHKeysAssignedToClusterOK) Error() string {
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

/*ListSSHKeysAssignedToClusterUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type ListSSHKeysAssignedToClusterUnauthorized struct {
}

func (o *ListSSHKeysAssignedToClusterUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/sshkeys][%d] listSshKeysAssignedToClusterUnauthorized ", 401)
}

func (o *ListSSHKeysAssignedToClusterUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListSSHKeysAssignedToClusterForbidden creates a ListSSHKeysAssignedToClusterForbidden with default headers values
func NewListSSHKeysAssignedToClusterForbidden() *ListSSHKeysAssignedToClusterForbidden {
	return &ListSSHKeysAssignedToClusterForbidden{}
}

/*ListSSHKeysAssignedToClusterForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type ListSSHKeysAssignedToClusterForbidden struct {
}

func (o *ListSSHKeysAssignedToClusterForbidden) Error() string {
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

/*ListSSHKeysAssignedToClusterDefault handles this case with default header values.

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

func (o *ListSSHKeysAssignedToClusterDefault) Error() string {
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
