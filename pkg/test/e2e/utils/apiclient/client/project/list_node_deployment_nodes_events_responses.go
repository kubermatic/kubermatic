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

// ListNodeDeploymentNodesEventsReader is a Reader for the ListNodeDeploymentNodesEvents structure.
type ListNodeDeploymentNodesEventsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListNodeDeploymentNodesEventsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListNodeDeploymentNodesEventsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListNodeDeploymentNodesEventsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListNodeDeploymentNodesEventsForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListNodeDeploymentNodesEventsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListNodeDeploymentNodesEventsOK creates a ListNodeDeploymentNodesEventsOK with default headers values
func NewListNodeDeploymentNodesEventsOK() *ListNodeDeploymentNodesEventsOK {
	return &ListNodeDeploymentNodesEventsOK{}
}

/*
ListNodeDeploymentNodesEventsOK describes a response with status code 200, with default header values.

Event
*/
type ListNodeDeploymentNodesEventsOK struct {
	Payload []*models.Event
}

// IsSuccess returns true when this list node deployment nodes events o k response has a 2xx status code
func (o *ListNodeDeploymentNodesEventsOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list node deployment nodes events o k response has a 3xx status code
func (o *ListNodeDeploymentNodesEventsOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list node deployment nodes events o k response has a 4xx status code
func (o *ListNodeDeploymentNodesEventsOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list node deployment nodes events o k response has a 5xx status code
func (o *ListNodeDeploymentNodesEventsOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list node deployment nodes events o k response a status code equal to that given
func (o *ListNodeDeploymentNodesEventsOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListNodeDeploymentNodesEventsOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}/nodes/events][%d] listNodeDeploymentNodesEventsOK  %+v", 200, o.Payload)
}

func (o *ListNodeDeploymentNodesEventsOK) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}/nodes/events][%d] listNodeDeploymentNodesEventsOK  %+v", 200, o.Payload)
}

func (o *ListNodeDeploymentNodesEventsOK) GetPayload() []*models.Event {
	return o.Payload
}

func (o *ListNodeDeploymentNodesEventsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListNodeDeploymentNodesEventsUnauthorized creates a ListNodeDeploymentNodesEventsUnauthorized with default headers values
func NewListNodeDeploymentNodesEventsUnauthorized() *ListNodeDeploymentNodesEventsUnauthorized {
	return &ListNodeDeploymentNodesEventsUnauthorized{}
}

/*
ListNodeDeploymentNodesEventsUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type ListNodeDeploymentNodesEventsUnauthorized struct {
}

// IsSuccess returns true when this list node deployment nodes events unauthorized response has a 2xx status code
func (o *ListNodeDeploymentNodesEventsUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list node deployment nodes events unauthorized response has a 3xx status code
func (o *ListNodeDeploymentNodesEventsUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list node deployment nodes events unauthorized response has a 4xx status code
func (o *ListNodeDeploymentNodesEventsUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this list node deployment nodes events unauthorized response has a 5xx status code
func (o *ListNodeDeploymentNodesEventsUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this list node deployment nodes events unauthorized response a status code equal to that given
func (o *ListNodeDeploymentNodesEventsUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *ListNodeDeploymentNodesEventsUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}/nodes/events][%d] listNodeDeploymentNodesEventsUnauthorized ", 401)
}

func (o *ListNodeDeploymentNodesEventsUnauthorized) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}/nodes/events][%d] listNodeDeploymentNodesEventsUnauthorized ", 401)
}

func (o *ListNodeDeploymentNodesEventsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListNodeDeploymentNodesEventsForbidden creates a ListNodeDeploymentNodesEventsForbidden with default headers values
func NewListNodeDeploymentNodesEventsForbidden() *ListNodeDeploymentNodesEventsForbidden {
	return &ListNodeDeploymentNodesEventsForbidden{}
}

/*
ListNodeDeploymentNodesEventsForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type ListNodeDeploymentNodesEventsForbidden struct {
}

// IsSuccess returns true when this list node deployment nodes events forbidden response has a 2xx status code
func (o *ListNodeDeploymentNodesEventsForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list node deployment nodes events forbidden response has a 3xx status code
func (o *ListNodeDeploymentNodesEventsForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list node deployment nodes events forbidden response has a 4xx status code
func (o *ListNodeDeploymentNodesEventsForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this list node deployment nodes events forbidden response has a 5xx status code
func (o *ListNodeDeploymentNodesEventsForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this list node deployment nodes events forbidden response a status code equal to that given
func (o *ListNodeDeploymentNodesEventsForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *ListNodeDeploymentNodesEventsForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}/nodes/events][%d] listNodeDeploymentNodesEventsForbidden ", 403)
}

func (o *ListNodeDeploymentNodesEventsForbidden) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}/nodes/events][%d] listNodeDeploymentNodesEventsForbidden ", 403)
}

func (o *ListNodeDeploymentNodesEventsForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListNodeDeploymentNodesEventsDefault creates a ListNodeDeploymentNodesEventsDefault with default headers values
func NewListNodeDeploymentNodesEventsDefault(code int) *ListNodeDeploymentNodesEventsDefault {
	return &ListNodeDeploymentNodesEventsDefault{
		_statusCode: code,
	}
}

/*
ListNodeDeploymentNodesEventsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListNodeDeploymentNodesEventsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list node deployment nodes events default response
func (o *ListNodeDeploymentNodesEventsDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list node deployment nodes events default response has a 2xx status code
func (o *ListNodeDeploymentNodesEventsDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list node deployment nodes events default response has a 3xx status code
func (o *ListNodeDeploymentNodesEventsDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list node deployment nodes events default response has a 4xx status code
func (o *ListNodeDeploymentNodesEventsDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list node deployment nodes events default response has a 5xx status code
func (o *ListNodeDeploymentNodesEventsDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list node deployment nodes events default response a status code equal to that given
func (o *ListNodeDeploymentNodesEventsDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListNodeDeploymentNodesEventsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}/nodes/events][%d] listNodeDeploymentNodesEvents default  %+v", o._statusCode, o.Payload)
}

func (o *ListNodeDeploymentNodesEventsDefault) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}/nodes/events][%d] listNodeDeploymentNodesEvents default  %+v", o._statusCode, o.Payload)
}

func (o *ListNodeDeploymentNodesEventsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListNodeDeploymentNodesEventsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
