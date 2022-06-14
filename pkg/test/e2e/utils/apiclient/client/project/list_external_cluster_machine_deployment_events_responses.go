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

// ListExternalClusterMachineDeploymentEventsReader is a Reader for the ListExternalClusterMachineDeploymentEvents structure.
type ListExternalClusterMachineDeploymentEventsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListExternalClusterMachineDeploymentEventsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListExternalClusterMachineDeploymentEventsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListExternalClusterMachineDeploymentEventsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListExternalClusterMachineDeploymentEventsForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListExternalClusterMachineDeploymentEventsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListExternalClusterMachineDeploymentEventsOK creates a ListExternalClusterMachineDeploymentEventsOK with default headers values
func NewListExternalClusterMachineDeploymentEventsOK() *ListExternalClusterMachineDeploymentEventsOK {
	return &ListExternalClusterMachineDeploymentEventsOK{}
}

/* ListExternalClusterMachineDeploymentEventsOK describes a response with status code 200, with default header values.

Event
*/
type ListExternalClusterMachineDeploymentEventsOK struct {
	Payload []*models.Event
}

func (o *ListExternalClusterMachineDeploymentEventsOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}/nodes/events][%d] listExternalClusterMachineDeploymentEventsOK  %+v", 200, o.Payload)
}
func (o *ListExternalClusterMachineDeploymentEventsOK) GetPayload() []*models.Event {
	return o.Payload
}

func (o *ListExternalClusterMachineDeploymentEventsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListExternalClusterMachineDeploymentEventsUnauthorized creates a ListExternalClusterMachineDeploymentEventsUnauthorized with default headers values
func NewListExternalClusterMachineDeploymentEventsUnauthorized() *ListExternalClusterMachineDeploymentEventsUnauthorized {
	return &ListExternalClusterMachineDeploymentEventsUnauthorized{}
}

/* ListExternalClusterMachineDeploymentEventsUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type ListExternalClusterMachineDeploymentEventsUnauthorized struct {
}

func (o *ListExternalClusterMachineDeploymentEventsUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}/nodes/events][%d] listExternalClusterMachineDeploymentEventsUnauthorized ", 401)
}

func (o *ListExternalClusterMachineDeploymentEventsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListExternalClusterMachineDeploymentEventsForbidden creates a ListExternalClusterMachineDeploymentEventsForbidden with default headers values
func NewListExternalClusterMachineDeploymentEventsForbidden() *ListExternalClusterMachineDeploymentEventsForbidden {
	return &ListExternalClusterMachineDeploymentEventsForbidden{}
}

/* ListExternalClusterMachineDeploymentEventsForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type ListExternalClusterMachineDeploymentEventsForbidden struct {
}

func (o *ListExternalClusterMachineDeploymentEventsForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}/nodes/events][%d] listExternalClusterMachineDeploymentEventsForbidden ", 403)
}

func (o *ListExternalClusterMachineDeploymentEventsForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListExternalClusterMachineDeploymentEventsDefault creates a ListExternalClusterMachineDeploymentEventsDefault with default headers values
func NewListExternalClusterMachineDeploymentEventsDefault(code int) *ListExternalClusterMachineDeploymentEventsDefault {
	return &ListExternalClusterMachineDeploymentEventsDefault{
		_statusCode: code,
	}
}

/* ListExternalClusterMachineDeploymentEventsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListExternalClusterMachineDeploymentEventsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list external cluster machine deployment events default response
func (o *ListExternalClusterMachineDeploymentEventsDefault) Code() int {
	return o._statusCode
}

func (o *ListExternalClusterMachineDeploymentEventsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}/nodes/events][%d] listExternalClusterMachineDeploymentEvents default  %+v", o._statusCode, o.Payload)
}
func (o *ListExternalClusterMachineDeploymentEventsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListExternalClusterMachineDeploymentEventsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
