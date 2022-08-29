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

// ListClusterRoleBindingReader is a Reader for the ListClusterRoleBinding structure.
type ListClusterRoleBindingReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListClusterRoleBindingReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListClusterRoleBindingOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListClusterRoleBindingUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListClusterRoleBindingForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListClusterRoleBindingDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListClusterRoleBindingOK creates a ListClusterRoleBindingOK with default headers values
func NewListClusterRoleBindingOK() *ListClusterRoleBindingOK {
	return &ListClusterRoleBindingOK{}
}

/*
ListClusterRoleBindingOK describes a response with status code 200, with default header values.

ClusterRoleBinding
*/
type ListClusterRoleBindingOK struct {
	Payload []*models.ClusterRoleBinding
}

// IsSuccess returns true when this list cluster role binding o k response has a 2xx status code
func (o *ListClusterRoleBindingOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list cluster role binding o k response has a 3xx status code
func (o *ListClusterRoleBindingOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list cluster role binding o k response has a 4xx status code
func (o *ListClusterRoleBindingOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list cluster role binding o k response has a 5xx status code
func (o *ListClusterRoleBindingOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list cluster role binding o k response a status code equal to that given
func (o *ListClusterRoleBindingOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListClusterRoleBindingOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterbindings][%d] listClusterRoleBindingOK  %+v", 200, o.Payload)
}

func (o *ListClusterRoleBindingOK) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterbindings][%d] listClusterRoleBindingOK  %+v", 200, o.Payload)
}

func (o *ListClusterRoleBindingOK) GetPayload() []*models.ClusterRoleBinding {
	return o.Payload
}

func (o *ListClusterRoleBindingOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListClusterRoleBindingUnauthorized creates a ListClusterRoleBindingUnauthorized with default headers values
func NewListClusterRoleBindingUnauthorized() *ListClusterRoleBindingUnauthorized {
	return &ListClusterRoleBindingUnauthorized{}
}

/*
ListClusterRoleBindingUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type ListClusterRoleBindingUnauthorized struct {
}

// IsSuccess returns true when this list cluster role binding unauthorized response has a 2xx status code
func (o *ListClusterRoleBindingUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list cluster role binding unauthorized response has a 3xx status code
func (o *ListClusterRoleBindingUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list cluster role binding unauthorized response has a 4xx status code
func (o *ListClusterRoleBindingUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this list cluster role binding unauthorized response has a 5xx status code
func (o *ListClusterRoleBindingUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this list cluster role binding unauthorized response a status code equal to that given
func (o *ListClusterRoleBindingUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *ListClusterRoleBindingUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterbindings][%d] listClusterRoleBindingUnauthorized ", 401)
}

func (o *ListClusterRoleBindingUnauthorized) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterbindings][%d] listClusterRoleBindingUnauthorized ", 401)
}

func (o *ListClusterRoleBindingUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListClusterRoleBindingForbidden creates a ListClusterRoleBindingForbidden with default headers values
func NewListClusterRoleBindingForbidden() *ListClusterRoleBindingForbidden {
	return &ListClusterRoleBindingForbidden{}
}

/*
ListClusterRoleBindingForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type ListClusterRoleBindingForbidden struct {
}

// IsSuccess returns true when this list cluster role binding forbidden response has a 2xx status code
func (o *ListClusterRoleBindingForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list cluster role binding forbidden response has a 3xx status code
func (o *ListClusterRoleBindingForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list cluster role binding forbidden response has a 4xx status code
func (o *ListClusterRoleBindingForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this list cluster role binding forbidden response has a 5xx status code
func (o *ListClusterRoleBindingForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this list cluster role binding forbidden response a status code equal to that given
func (o *ListClusterRoleBindingForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *ListClusterRoleBindingForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterbindings][%d] listClusterRoleBindingForbidden ", 403)
}

func (o *ListClusterRoleBindingForbidden) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterbindings][%d] listClusterRoleBindingForbidden ", 403)
}

func (o *ListClusterRoleBindingForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListClusterRoleBindingDefault creates a ListClusterRoleBindingDefault with default headers values
func NewListClusterRoleBindingDefault(code int) *ListClusterRoleBindingDefault {
	return &ListClusterRoleBindingDefault{
		_statusCode: code,
	}
}

/*
ListClusterRoleBindingDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListClusterRoleBindingDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list cluster role binding default response
func (o *ListClusterRoleBindingDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list cluster role binding default response has a 2xx status code
func (o *ListClusterRoleBindingDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list cluster role binding default response has a 3xx status code
func (o *ListClusterRoleBindingDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list cluster role binding default response has a 4xx status code
func (o *ListClusterRoleBindingDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list cluster role binding default response has a 5xx status code
func (o *ListClusterRoleBindingDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list cluster role binding default response a status code equal to that given
func (o *ListClusterRoleBindingDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListClusterRoleBindingDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterbindings][%d] listClusterRoleBinding default  %+v", o._statusCode, o.Payload)
}

func (o *ListClusterRoleBindingDefault) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/clusterbindings][%d] listClusterRoleBinding default  %+v", o._statusCode, o.Payload)
}

func (o *ListClusterRoleBindingDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListClusterRoleBindingDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
