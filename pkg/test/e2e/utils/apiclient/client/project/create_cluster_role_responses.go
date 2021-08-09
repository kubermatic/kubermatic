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

// CreateClusterRoleReader is a Reader for the CreateClusterRole structure.
type CreateClusterRoleReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *CreateClusterRoleReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 201:
		result := NewCreateClusterRoleCreated()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewCreateClusterRoleUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewCreateClusterRoleForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewCreateClusterRoleDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewCreateClusterRoleCreated creates a CreateClusterRoleCreated with default headers values
func NewCreateClusterRoleCreated() *CreateClusterRoleCreated {
	return &CreateClusterRoleCreated{}
}

/* CreateClusterRoleCreated describes a response with status code 201, with default header values.

ClusterRole
*/
type CreateClusterRoleCreated struct {
	Payload *models.ClusterRole
}

func (o *CreateClusterRoleCreated) Error() string {
	return fmt.Sprintf("[POST /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterroles][%d] createClusterRoleCreated  %+v", 201, o.Payload)
}
func (o *CreateClusterRoleCreated) GetPayload() *models.ClusterRole {
	return o.Payload
}

func (o *CreateClusterRoleCreated) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ClusterRole)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewCreateClusterRoleUnauthorized creates a CreateClusterRoleUnauthorized with default headers values
func NewCreateClusterRoleUnauthorized() *CreateClusterRoleUnauthorized {
	return &CreateClusterRoleUnauthorized{}
}

/* CreateClusterRoleUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type CreateClusterRoleUnauthorized struct {
}

func (o *CreateClusterRoleUnauthorized) Error() string {
	return fmt.Sprintf("[POST /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterroles][%d] createClusterRoleUnauthorized ", 401)
}

func (o *CreateClusterRoleUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateClusterRoleForbidden creates a CreateClusterRoleForbidden with default headers values
func NewCreateClusterRoleForbidden() *CreateClusterRoleForbidden {
	return &CreateClusterRoleForbidden{}
}

/* CreateClusterRoleForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type CreateClusterRoleForbidden struct {
}

func (o *CreateClusterRoleForbidden) Error() string {
	return fmt.Sprintf("[POST /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterroles][%d] createClusterRoleForbidden ", 403)
}

func (o *CreateClusterRoleForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateClusterRoleDefault creates a CreateClusterRoleDefault with default headers values
func NewCreateClusterRoleDefault(code int) *CreateClusterRoleDefault {
	return &CreateClusterRoleDefault{
		_statusCode: code,
	}
}

/* CreateClusterRoleDefault describes a response with status code -1, with default header values.

errorResponse
*/
type CreateClusterRoleDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the create cluster role default response
func (o *CreateClusterRoleDefault) Code() int {
	return o._statusCode
}

func (o *CreateClusterRoleDefault) Error() string {
	return fmt.Sprintf("[POST /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterroles][%d] createClusterRole default  %+v", o._statusCode, o.Payload)
}
func (o *CreateClusterRoleDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *CreateClusterRoleDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
