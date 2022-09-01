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

// DeleteMachineDeploymentNodeReader is a Reader for the DeleteMachineDeploymentNode structure.
type DeleteMachineDeploymentNodeReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeleteMachineDeploymentNodeReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewDeleteMachineDeploymentNodeOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewDeleteMachineDeploymentNodeUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewDeleteMachineDeploymentNodeForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewDeleteMachineDeploymentNodeDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewDeleteMachineDeploymentNodeOK creates a DeleteMachineDeploymentNodeOK with default headers values
func NewDeleteMachineDeploymentNodeOK() *DeleteMachineDeploymentNodeOK {
	return &DeleteMachineDeploymentNodeOK{}
}

/* DeleteMachineDeploymentNodeOK describes a response with status code 200, with default header values.

EmptyResponse is a empty response
*/
type DeleteMachineDeploymentNodeOK struct {
}

// IsSuccess returns true when this delete machine deployment node o k response has a 2xx status code
func (o *DeleteMachineDeploymentNodeOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this delete machine deployment node o k response has a 3xx status code
func (o *DeleteMachineDeploymentNodeOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this delete machine deployment node o k response has a 4xx status code
func (o *DeleteMachineDeploymentNodeOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this delete machine deployment node o k response has a 5xx status code
func (o *DeleteMachineDeploymentNodeOK) IsServerError() bool {
	return false
}

// IsCode returns true when this delete machine deployment node o k response a status code equal to that given
func (o *DeleteMachineDeploymentNodeOK) IsCode(code int) bool {
	return code == 200
}

func (o *DeleteMachineDeploymentNodeOK) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments/nodes/{node_id}][%d] deleteMachineDeploymentNodeOK ", 200)
}

func (o *DeleteMachineDeploymentNodeOK) String() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments/nodes/{node_id}][%d] deleteMachineDeploymentNodeOK ", 200)
}

func (o *DeleteMachineDeploymentNodeOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteMachineDeploymentNodeUnauthorized creates a DeleteMachineDeploymentNodeUnauthorized with default headers values
func NewDeleteMachineDeploymentNodeUnauthorized() *DeleteMachineDeploymentNodeUnauthorized {
	return &DeleteMachineDeploymentNodeUnauthorized{}
}

/* DeleteMachineDeploymentNodeUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type DeleteMachineDeploymentNodeUnauthorized struct {
}

// IsSuccess returns true when this delete machine deployment node unauthorized response has a 2xx status code
func (o *DeleteMachineDeploymentNodeUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this delete machine deployment node unauthorized response has a 3xx status code
func (o *DeleteMachineDeploymentNodeUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this delete machine deployment node unauthorized response has a 4xx status code
func (o *DeleteMachineDeploymentNodeUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this delete machine deployment node unauthorized response has a 5xx status code
func (o *DeleteMachineDeploymentNodeUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this delete machine deployment node unauthorized response a status code equal to that given
func (o *DeleteMachineDeploymentNodeUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *DeleteMachineDeploymentNodeUnauthorized) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments/nodes/{node_id}][%d] deleteMachineDeploymentNodeUnauthorized ", 401)
}

func (o *DeleteMachineDeploymentNodeUnauthorized) String() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments/nodes/{node_id}][%d] deleteMachineDeploymentNodeUnauthorized ", 401)
}

func (o *DeleteMachineDeploymentNodeUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteMachineDeploymentNodeForbidden creates a DeleteMachineDeploymentNodeForbidden with default headers values
func NewDeleteMachineDeploymentNodeForbidden() *DeleteMachineDeploymentNodeForbidden {
	return &DeleteMachineDeploymentNodeForbidden{}
}

/* DeleteMachineDeploymentNodeForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type DeleteMachineDeploymentNodeForbidden struct {
}

// IsSuccess returns true when this delete machine deployment node forbidden response has a 2xx status code
func (o *DeleteMachineDeploymentNodeForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this delete machine deployment node forbidden response has a 3xx status code
func (o *DeleteMachineDeploymentNodeForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this delete machine deployment node forbidden response has a 4xx status code
func (o *DeleteMachineDeploymentNodeForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this delete machine deployment node forbidden response has a 5xx status code
func (o *DeleteMachineDeploymentNodeForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this delete machine deployment node forbidden response a status code equal to that given
func (o *DeleteMachineDeploymentNodeForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *DeleteMachineDeploymentNodeForbidden) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments/nodes/{node_id}][%d] deleteMachineDeploymentNodeForbidden ", 403)
}

func (o *DeleteMachineDeploymentNodeForbidden) String() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments/nodes/{node_id}][%d] deleteMachineDeploymentNodeForbidden ", 403)
}

func (o *DeleteMachineDeploymentNodeForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteMachineDeploymentNodeDefault creates a DeleteMachineDeploymentNodeDefault with default headers values
func NewDeleteMachineDeploymentNodeDefault(code int) *DeleteMachineDeploymentNodeDefault {
	return &DeleteMachineDeploymentNodeDefault{
		_statusCode: code,
	}
}

/* DeleteMachineDeploymentNodeDefault describes a response with status code -1, with default header values.

errorResponse
*/
type DeleteMachineDeploymentNodeDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the delete machine deployment node default response
func (o *DeleteMachineDeploymentNodeDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this delete machine deployment node default response has a 2xx status code
func (o *DeleteMachineDeploymentNodeDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this delete machine deployment node default response has a 3xx status code
func (o *DeleteMachineDeploymentNodeDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this delete machine deployment node default response has a 4xx status code
func (o *DeleteMachineDeploymentNodeDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this delete machine deployment node default response has a 5xx status code
func (o *DeleteMachineDeploymentNodeDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this delete machine deployment node default response a status code equal to that given
func (o *DeleteMachineDeploymentNodeDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *DeleteMachineDeploymentNodeDefault) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments/nodes/{node_id}][%d] deleteMachineDeploymentNode default  %+v", o._statusCode, o.Payload)
}

func (o *DeleteMachineDeploymentNodeDefault) String() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments/nodes/{node_id}][%d] deleteMachineDeploymentNode default  %+v", o._statusCode, o.Payload)
}

func (o *DeleteMachineDeploymentNodeDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *DeleteMachineDeploymentNodeDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
