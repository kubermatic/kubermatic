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

// GetExternalClusterMachineDeploymentReader is a Reader for the GetExternalClusterMachineDeployment structure.
type GetExternalClusterMachineDeploymentReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetExternalClusterMachineDeploymentReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetExternalClusterMachineDeploymentOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewGetExternalClusterMachineDeploymentUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewGetExternalClusterMachineDeploymentForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewGetExternalClusterMachineDeploymentDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetExternalClusterMachineDeploymentOK creates a GetExternalClusterMachineDeploymentOK with default headers values
func NewGetExternalClusterMachineDeploymentOK() *GetExternalClusterMachineDeploymentOK {
	return &GetExternalClusterMachineDeploymentOK{}
}

/* GetExternalClusterMachineDeploymentOK describes a response with status code 200, with default header values.

ExternalClusterMachineDeployment
*/
type GetExternalClusterMachineDeploymentOK struct {
	Payload *models.ExternalClusterMachineDeployment
}

// IsSuccess returns true when this get external cluster machine deployment o k response has a 2xx status code
func (o *GetExternalClusterMachineDeploymentOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this get external cluster machine deployment o k response has a 3xx status code
func (o *GetExternalClusterMachineDeploymentOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get external cluster machine deployment o k response has a 4xx status code
func (o *GetExternalClusterMachineDeploymentOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this get external cluster machine deployment o k response has a 5xx status code
func (o *GetExternalClusterMachineDeploymentOK) IsServerError() bool {
	return false
}

// IsCode returns true when this get external cluster machine deployment o k response a status code equal to that given
func (o *GetExternalClusterMachineDeploymentOK) IsCode(code int) bool {
	return code == 200
}

func (o *GetExternalClusterMachineDeploymentOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}][%d] getExternalClusterMachineDeploymentOK  %+v", 200, o.Payload)
}

func (o *GetExternalClusterMachineDeploymentOK) String() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}][%d] getExternalClusterMachineDeploymentOK  %+v", 200, o.Payload)
}

func (o *GetExternalClusterMachineDeploymentOK) GetPayload() *models.ExternalClusterMachineDeployment {
	return o.Payload
}

func (o *GetExternalClusterMachineDeploymentOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ExternalClusterMachineDeployment)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetExternalClusterMachineDeploymentUnauthorized creates a GetExternalClusterMachineDeploymentUnauthorized with default headers values
func NewGetExternalClusterMachineDeploymentUnauthorized() *GetExternalClusterMachineDeploymentUnauthorized {
	return &GetExternalClusterMachineDeploymentUnauthorized{}
}

/* GetExternalClusterMachineDeploymentUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type GetExternalClusterMachineDeploymentUnauthorized struct {
}

// IsSuccess returns true when this get external cluster machine deployment unauthorized response has a 2xx status code
func (o *GetExternalClusterMachineDeploymentUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this get external cluster machine deployment unauthorized response has a 3xx status code
func (o *GetExternalClusterMachineDeploymentUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get external cluster machine deployment unauthorized response has a 4xx status code
func (o *GetExternalClusterMachineDeploymentUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this get external cluster machine deployment unauthorized response has a 5xx status code
func (o *GetExternalClusterMachineDeploymentUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this get external cluster machine deployment unauthorized response a status code equal to that given
func (o *GetExternalClusterMachineDeploymentUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *GetExternalClusterMachineDeploymentUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}][%d] getExternalClusterMachineDeploymentUnauthorized ", 401)
}

func (o *GetExternalClusterMachineDeploymentUnauthorized) String() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}][%d] getExternalClusterMachineDeploymentUnauthorized ", 401)
}

func (o *GetExternalClusterMachineDeploymentUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetExternalClusterMachineDeploymentForbidden creates a GetExternalClusterMachineDeploymentForbidden with default headers values
func NewGetExternalClusterMachineDeploymentForbidden() *GetExternalClusterMachineDeploymentForbidden {
	return &GetExternalClusterMachineDeploymentForbidden{}
}

/* GetExternalClusterMachineDeploymentForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type GetExternalClusterMachineDeploymentForbidden struct {
}

// IsSuccess returns true when this get external cluster machine deployment forbidden response has a 2xx status code
func (o *GetExternalClusterMachineDeploymentForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this get external cluster machine deployment forbidden response has a 3xx status code
func (o *GetExternalClusterMachineDeploymentForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get external cluster machine deployment forbidden response has a 4xx status code
func (o *GetExternalClusterMachineDeploymentForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this get external cluster machine deployment forbidden response has a 5xx status code
func (o *GetExternalClusterMachineDeploymentForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this get external cluster machine deployment forbidden response a status code equal to that given
func (o *GetExternalClusterMachineDeploymentForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *GetExternalClusterMachineDeploymentForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}][%d] getExternalClusterMachineDeploymentForbidden ", 403)
}

func (o *GetExternalClusterMachineDeploymentForbidden) String() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}][%d] getExternalClusterMachineDeploymentForbidden ", 403)
}

func (o *GetExternalClusterMachineDeploymentForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetExternalClusterMachineDeploymentDefault creates a GetExternalClusterMachineDeploymentDefault with default headers values
func NewGetExternalClusterMachineDeploymentDefault(code int) *GetExternalClusterMachineDeploymentDefault {
	return &GetExternalClusterMachineDeploymentDefault{
		_statusCode: code,
	}
}

/* GetExternalClusterMachineDeploymentDefault describes a response with status code -1, with default header values.

errorResponse
*/
type GetExternalClusterMachineDeploymentDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the get external cluster machine deployment default response
func (o *GetExternalClusterMachineDeploymentDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this get external cluster machine deployment default response has a 2xx status code
func (o *GetExternalClusterMachineDeploymentDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this get external cluster machine deployment default response has a 3xx status code
func (o *GetExternalClusterMachineDeploymentDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this get external cluster machine deployment default response has a 4xx status code
func (o *GetExternalClusterMachineDeploymentDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this get external cluster machine deployment default response has a 5xx status code
func (o *GetExternalClusterMachineDeploymentDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this get external cluster machine deployment default response a status code equal to that given
func (o *GetExternalClusterMachineDeploymentDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *GetExternalClusterMachineDeploymentDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}][%d] getExternalClusterMachineDeployment default  %+v", o._statusCode, o.Payload)
}

func (o *GetExternalClusterMachineDeploymentDefault) String() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}][%d] getExternalClusterMachineDeployment default  %+v", o._statusCode, o.Payload)
}

func (o *GetExternalClusterMachineDeploymentDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *GetExternalClusterMachineDeploymentDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
