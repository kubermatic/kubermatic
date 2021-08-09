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

// ListSSHKeysAssignedToClusterV2Reader is a Reader for the ListSSHKeysAssignedToClusterV2 structure.
type ListSSHKeysAssignedToClusterV2Reader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListSSHKeysAssignedToClusterV2Reader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListSSHKeysAssignedToClusterV2OK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListSSHKeysAssignedToClusterV2Unauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListSSHKeysAssignedToClusterV2Forbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListSSHKeysAssignedToClusterV2Default(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListSSHKeysAssignedToClusterV2OK creates a ListSSHKeysAssignedToClusterV2OK with default headers values
func NewListSSHKeysAssignedToClusterV2OK() *ListSSHKeysAssignedToClusterV2OK {
	return &ListSSHKeysAssignedToClusterV2OK{}
}

/* ListSSHKeysAssignedToClusterV2OK describes a response with status code 200, with default header values.

SSHKey
*/
type ListSSHKeysAssignedToClusterV2OK struct {
	Payload []*models.SSHKey
}

func (o *ListSSHKeysAssignedToClusterV2OK) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/sshkeys][%d] listSshKeysAssignedToClusterV2OK  %+v", 200, o.Payload)
}
func (o *ListSSHKeysAssignedToClusterV2OK) GetPayload() []*models.SSHKey {
	return o.Payload
}

func (o *ListSSHKeysAssignedToClusterV2OK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListSSHKeysAssignedToClusterV2Unauthorized creates a ListSSHKeysAssignedToClusterV2Unauthorized with default headers values
func NewListSSHKeysAssignedToClusterV2Unauthorized() *ListSSHKeysAssignedToClusterV2Unauthorized {
	return &ListSSHKeysAssignedToClusterV2Unauthorized{}
}

/* ListSSHKeysAssignedToClusterV2Unauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type ListSSHKeysAssignedToClusterV2Unauthorized struct {
}

func (o *ListSSHKeysAssignedToClusterV2Unauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/sshkeys][%d] listSshKeysAssignedToClusterV2Unauthorized ", 401)
}

func (o *ListSSHKeysAssignedToClusterV2Unauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListSSHKeysAssignedToClusterV2Forbidden creates a ListSSHKeysAssignedToClusterV2Forbidden with default headers values
func NewListSSHKeysAssignedToClusterV2Forbidden() *ListSSHKeysAssignedToClusterV2Forbidden {
	return &ListSSHKeysAssignedToClusterV2Forbidden{}
}

/* ListSSHKeysAssignedToClusterV2Forbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type ListSSHKeysAssignedToClusterV2Forbidden struct {
}

func (o *ListSSHKeysAssignedToClusterV2Forbidden) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/sshkeys][%d] listSshKeysAssignedToClusterV2Forbidden ", 403)
}

func (o *ListSSHKeysAssignedToClusterV2Forbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListSSHKeysAssignedToClusterV2Default creates a ListSSHKeysAssignedToClusterV2Default with default headers values
func NewListSSHKeysAssignedToClusterV2Default(code int) *ListSSHKeysAssignedToClusterV2Default {
	return &ListSSHKeysAssignedToClusterV2Default{
		_statusCode: code,
	}
}

/* ListSSHKeysAssignedToClusterV2Default describes a response with status code -1, with default header values.

errorResponse
*/
type ListSSHKeysAssignedToClusterV2Default struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list SSH keys assigned to cluster v2 default response
func (o *ListSSHKeysAssignedToClusterV2Default) Code() int {
	return o._statusCode
}

func (o *ListSSHKeysAssignedToClusterV2Default) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/sshkeys][%d] listSSHKeysAssignedToClusterV2 default  %+v", o._statusCode, o.Payload)
}
func (o *ListSSHKeysAssignedToClusterV2Default) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListSSHKeysAssignedToClusterV2Default) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
