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

// ListRoleReader is a Reader for the ListRole structure.
type ListRoleReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListRoleReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListRoleOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListRoleUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListRoleForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListRoleDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListRoleOK creates a ListRoleOK with default headers values
func NewListRoleOK() *ListRoleOK {
	return &ListRoleOK{}
}

/*ListRoleOK handles this case with default header values.

Role
*/
type ListRoleOK struct {
	Payload []*models.Role
}

func (o *ListRoleOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/roles][%d] listRoleOK  %+v", 200, o.Payload)
}

func (o *ListRoleOK) GetPayload() []*models.Role {
	return o.Payload
}

func (o *ListRoleOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListRoleUnauthorized creates a ListRoleUnauthorized with default headers values
func NewListRoleUnauthorized() *ListRoleUnauthorized {
	return &ListRoleUnauthorized{}
}

/*ListRoleUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type ListRoleUnauthorized struct {
}

func (o *ListRoleUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/roles][%d] listRoleUnauthorized ", 401)
}

func (o *ListRoleUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListRoleForbidden creates a ListRoleForbidden with default headers values
func NewListRoleForbidden() *ListRoleForbidden {
	return &ListRoleForbidden{}
}

/*ListRoleForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type ListRoleForbidden struct {
}

func (o *ListRoleForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/roles][%d] listRoleForbidden ", 403)
}

func (o *ListRoleForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListRoleDefault creates a ListRoleDefault with default headers values
func NewListRoleDefault(code int) *ListRoleDefault {
	return &ListRoleDefault{
		_statusCode: code,
	}
}

/*ListRoleDefault handles this case with default header values.

errorResponse
*/
type ListRoleDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list role default response
func (o *ListRoleDefault) Code() int {
	return o._statusCode
}

func (o *ListRoleDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/roles][%d] listRole default  %+v", o._statusCode, o.Payload)
}

func (o *ListRoleDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListRoleDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
