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

// DetachSSHKeyFromClusterV2Reader is a Reader for the DetachSSHKeyFromClusterV2 structure.
type DetachSSHKeyFromClusterV2Reader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DetachSSHKeyFromClusterV2Reader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewDetachSSHKeyFromClusterV2OK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewDetachSSHKeyFromClusterV2Unauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewDetachSSHKeyFromClusterV2Forbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewDetachSSHKeyFromClusterV2Default(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewDetachSSHKeyFromClusterV2OK creates a DetachSSHKeyFromClusterV2OK with default headers values
func NewDetachSSHKeyFromClusterV2OK() *DetachSSHKeyFromClusterV2OK {
	return &DetachSSHKeyFromClusterV2OK{}
}

/* DetachSSHKeyFromClusterV2OK describes a response with status code 200, with default header values.

EmptyResponse is a empty response
*/
type DetachSSHKeyFromClusterV2OK struct {
}

func (o *DetachSSHKeyFromClusterV2OK) Error() string {
	return fmt.Sprintf("[DELETE /api/projects/{project_id}/clusters/{cluster_id}/sshkeys/{key_id}][%d] detachSshKeyFromClusterV2OK ", 200)
}

func (o *DetachSSHKeyFromClusterV2OK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDetachSSHKeyFromClusterV2Unauthorized creates a DetachSSHKeyFromClusterV2Unauthorized with default headers values
func NewDetachSSHKeyFromClusterV2Unauthorized() *DetachSSHKeyFromClusterV2Unauthorized {
	return &DetachSSHKeyFromClusterV2Unauthorized{}
}

/* DetachSSHKeyFromClusterV2Unauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type DetachSSHKeyFromClusterV2Unauthorized struct {
}

func (o *DetachSSHKeyFromClusterV2Unauthorized) Error() string {
	return fmt.Sprintf("[DELETE /api/projects/{project_id}/clusters/{cluster_id}/sshkeys/{key_id}][%d] detachSshKeyFromClusterV2Unauthorized ", 401)
}

func (o *DetachSSHKeyFromClusterV2Unauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDetachSSHKeyFromClusterV2Forbidden creates a DetachSSHKeyFromClusterV2Forbidden with default headers values
func NewDetachSSHKeyFromClusterV2Forbidden() *DetachSSHKeyFromClusterV2Forbidden {
	return &DetachSSHKeyFromClusterV2Forbidden{}
}

/* DetachSSHKeyFromClusterV2Forbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type DetachSSHKeyFromClusterV2Forbidden struct {
}

func (o *DetachSSHKeyFromClusterV2Forbidden) Error() string {
	return fmt.Sprintf("[DELETE /api/projects/{project_id}/clusters/{cluster_id}/sshkeys/{key_id}][%d] detachSshKeyFromClusterV2Forbidden ", 403)
}

func (o *DetachSSHKeyFromClusterV2Forbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDetachSSHKeyFromClusterV2Default creates a DetachSSHKeyFromClusterV2Default with default headers values
func NewDetachSSHKeyFromClusterV2Default(code int) *DetachSSHKeyFromClusterV2Default {
	return &DetachSSHKeyFromClusterV2Default{
		_statusCode: code,
	}
}

/* DetachSSHKeyFromClusterV2Default describes a response with status code -1, with default header values.

errorResponse
*/
type DetachSSHKeyFromClusterV2Default struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the detach SSH key from cluster v2 default response
func (o *DetachSSHKeyFromClusterV2Default) Code() int {
	return o._statusCode
}

func (o *DetachSSHKeyFromClusterV2Default) Error() string {
	return fmt.Sprintf("[DELETE /api/projects/{project_id}/clusters/{cluster_id}/sshkeys/{key_id}][%d] detachSSHKeyFromClusterV2 default  %+v", o._statusCode, o.Payload)
}
func (o *DetachSSHKeyFromClusterV2Default) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *DetachSSHKeyFromClusterV2Default) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
