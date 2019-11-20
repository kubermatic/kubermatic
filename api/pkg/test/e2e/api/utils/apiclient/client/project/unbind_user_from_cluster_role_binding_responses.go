// Code generated by go-swagger; DO NOT EDIT.

package project

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	models "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// UnbindUserFromClusterRoleBindingReader is a Reader for the UnbindUserFromClusterRoleBinding structure.
type UnbindUserFromClusterRoleBindingReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *UnbindUserFromClusterRoleBindingReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 201:
		result := NewUnbindUserFromClusterRoleBindingCreated()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	case 401:
		result := NewUnbindUserFromClusterRoleBindingUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result

	case 403:
		result := NewUnbindUserFromClusterRoleBindingForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result

	default:
		result := NewUnbindUserFromClusterRoleBindingDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewUnbindUserFromClusterRoleBindingCreated creates a UnbindUserFromClusterRoleBindingCreated with default headers values
func NewUnbindUserFromClusterRoleBindingCreated() *UnbindUserFromClusterRoleBindingCreated {
	return &UnbindUserFromClusterRoleBindingCreated{}
}

/*UnbindUserFromClusterRoleBindingCreated handles this case with default header values.

ClusterRoleBinding
*/
type UnbindUserFromClusterRoleBindingCreated struct {
	Payload *models.ClusterRoleBinding
}

func (o *UnbindUserFromClusterRoleBindingCreated) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterroles/{role_id}/clusterbindings][%d] unbindUserFromClusterRoleBindingCreated  %+v", 201, o.Payload)
}

func (o *UnbindUserFromClusterRoleBindingCreated) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ClusterRoleBinding)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewUnbindUserFromClusterRoleBindingUnauthorized creates a UnbindUserFromClusterRoleBindingUnauthorized with default headers values
func NewUnbindUserFromClusterRoleBindingUnauthorized() *UnbindUserFromClusterRoleBindingUnauthorized {
	return &UnbindUserFromClusterRoleBindingUnauthorized{}
}

/*UnbindUserFromClusterRoleBindingUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type UnbindUserFromClusterRoleBindingUnauthorized struct {
}

func (o *UnbindUserFromClusterRoleBindingUnauthorized) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterroles/{role_id}/clusterbindings][%d] unbindUserFromClusterRoleBindingUnauthorized ", 401)
}

func (o *UnbindUserFromClusterRoleBindingUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewUnbindUserFromClusterRoleBindingForbidden creates a UnbindUserFromClusterRoleBindingForbidden with default headers values
func NewUnbindUserFromClusterRoleBindingForbidden() *UnbindUserFromClusterRoleBindingForbidden {
	return &UnbindUserFromClusterRoleBindingForbidden{}
}

/*UnbindUserFromClusterRoleBindingForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type UnbindUserFromClusterRoleBindingForbidden struct {
}

func (o *UnbindUserFromClusterRoleBindingForbidden) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterroles/{role_id}/clusterbindings][%d] unbindUserFromClusterRoleBindingForbidden ", 403)
}

func (o *UnbindUserFromClusterRoleBindingForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewUnbindUserFromClusterRoleBindingDefault creates a UnbindUserFromClusterRoleBindingDefault with default headers values
func NewUnbindUserFromClusterRoleBindingDefault(code int) *UnbindUserFromClusterRoleBindingDefault {
	return &UnbindUserFromClusterRoleBindingDefault{
		_statusCode: code,
	}
}

/*UnbindUserFromClusterRoleBindingDefault handles this case with default header values.

errorResponse
*/
type UnbindUserFromClusterRoleBindingDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the unbind user from cluster role binding default response
func (o *UnbindUserFromClusterRoleBindingDefault) Code() int {
	return o._statusCode
}

func (o *UnbindUserFromClusterRoleBindingDefault) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterroles/{role_id}/clusterbindings][%d] unbindUserFromClusterRoleBinding default  %+v", o._statusCode, o.Payload)
}

func (o *UnbindUserFromClusterRoleBindingDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
