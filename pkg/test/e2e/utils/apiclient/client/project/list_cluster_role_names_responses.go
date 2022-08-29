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

// ListClusterRoleNamesReader is a Reader for the ListClusterRoleNames structure.
type ListClusterRoleNamesReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListClusterRoleNamesReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListClusterRoleNamesOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListClusterRoleNamesUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListClusterRoleNamesForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListClusterRoleNamesDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListClusterRoleNamesOK creates a ListClusterRoleNamesOK with default headers values
func NewListClusterRoleNamesOK() *ListClusterRoleNamesOK {
	return &ListClusterRoleNamesOK{}
}

/*
ListClusterRoleNamesOK describes a response with status code 200, with default header values.

ClusterRoleName
*/
type ListClusterRoleNamesOK struct {
	Payload []*models.ClusterRoleName
}

// IsSuccess returns true when this list cluster role names o k response has a 2xx status code
func (o *ListClusterRoleNamesOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list cluster role names o k response has a 3xx status code
func (o *ListClusterRoleNamesOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list cluster role names o k response has a 4xx status code
func (o *ListClusterRoleNamesOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list cluster role names o k response has a 5xx status code
func (o *ListClusterRoleNamesOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list cluster role names o k response a status code equal to that given
func (o *ListClusterRoleNamesOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListClusterRoleNamesOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterrolenames][%d] listClusterRoleNamesOK  %+v", 200, o.Payload)
}

func (o *ListClusterRoleNamesOK) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterrolenames][%d] listClusterRoleNamesOK  %+v", 200, o.Payload)
}

func (o *ListClusterRoleNamesOK) GetPayload() []*models.ClusterRoleName {
	return o.Payload
}

func (o *ListClusterRoleNamesOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListClusterRoleNamesUnauthorized creates a ListClusterRoleNamesUnauthorized with default headers values
func NewListClusterRoleNamesUnauthorized() *ListClusterRoleNamesUnauthorized {
	return &ListClusterRoleNamesUnauthorized{}
}

/*
ListClusterRoleNamesUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type ListClusterRoleNamesUnauthorized struct {
}

// IsSuccess returns true when this list cluster role names unauthorized response has a 2xx status code
func (o *ListClusterRoleNamesUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list cluster role names unauthorized response has a 3xx status code
func (o *ListClusterRoleNamesUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list cluster role names unauthorized response has a 4xx status code
func (o *ListClusterRoleNamesUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this list cluster role names unauthorized response has a 5xx status code
func (o *ListClusterRoleNamesUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this list cluster role names unauthorized response a status code equal to that given
func (o *ListClusterRoleNamesUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *ListClusterRoleNamesUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterrolenames][%d] listClusterRoleNamesUnauthorized ", 401)
}

func (o *ListClusterRoleNamesUnauthorized) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterrolenames][%d] listClusterRoleNamesUnauthorized ", 401)
}

func (o *ListClusterRoleNamesUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListClusterRoleNamesForbidden creates a ListClusterRoleNamesForbidden with default headers values
func NewListClusterRoleNamesForbidden() *ListClusterRoleNamesForbidden {
	return &ListClusterRoleNamesForbidden{}
}

/*
ListClusterRoleNamesForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type ListClusterRoleNamesForbidden struct {
}

// IsSuccess returns true when this list cluster role names forbidden response has a 2xx status code
func (o *ListClusterRoleNamesForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list cluster role names forbidden response has a 3xx status code
func (o *ListClusterRoleNamesForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list cluster role names forbidden response has a 4xx status code
func (o *ListClusterRoleNamesForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this list cluster role names forbidden response has a 5xx status code
func (o *ListClusterRoleNamesForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this list cluster role names forbidden response a status code equal to that given
func (o *ListClusterRoleNamesForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *ListClusterRoleNamesForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterrolenames][%d] listClusterRoleNamesForbidden ", 403)
}

func (o *ListClusterRoleNamesForbidden) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterrolenames][%d] listClusterRoleNamesForbidden ", 403)
}

func (o *ListClusterRoleNamesForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListClusterRoleNamesDefault creates a ListClusterRoleNamesDefault with default headers values
func NewListClusterRoleNamesDefault(code int) *ListClusterRoleNamesDefault {
	return &ListClusterRoleNamesDefault{
		_statusCode: code,
	}
}

/*
ListClusterRoleNamesDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListClusterRoleNamesDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list cluster role names default response
func (o *ListClusterRoleNamesDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list cluster role names default response has a 2xx status code
func (o *ListClusterRoleNamesDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list cluster role names default response has a 3xx status code
func (o *ListClusterRoleNamesDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list cluster role names default response has a 4xx status code
func (o *ListClusterRoleNamesDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list cluster role names default response has a 5xx status code
func (o *ListClusterRoleNamesDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list cluster role names default response a status code equal to that given
func (o *ListClusterRoleNamesDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListClusterRoleNamesDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterrolenames][%d] listClusterRoleNames default  %+v", o._statusCode, o.Payload)
}

func (o *ListClusterRoleNamesDefault) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterrolenames][%d] listClusterRoleNames default  %+v", o._statusCode, o.Payload)
}

func (o *ListClusterRoleNamesDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListClusterRoleNamesDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
