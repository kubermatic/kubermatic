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

// UpgradeClusterNodeDeploymentsReader is a Reader for the UpgradeClusterNodeDeployments structure.
type UpgradeClusterNodeDeploymentsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *UpgradeClusterNodeDeploymentsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewUpgradeClusterNodeDeploymentsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewUpgradeClusterNodeDeploymentsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewUpgradeClusterNodeDeploymentsForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewUpgradeClusterNodeDeploymentsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewUpgradeClusterNodeDeploymentsOK creates a UpgradeClusterNodeDeploymentsOK with default headers values
func NewUpgradeClusterNodeDeploymentsOK() *UpgradeClusterNodeDeploymentsOK {
	return &UpgradeClusterNodeDeploymentsOK{}
}

/*UpgradeClusterNodeDeploymentsOK handles this case with default header values.

EmptyResponse is a empty response
*/
type UpgradeClusterNodeDeploymentsOK struct {
}

func (o *UpgradeClusterNodeDeploymentsOK) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/upgrades][%d] upgradeClusterNodeDeploymentsOK ", 200)
}

func (o *UpgradeClusterNodeDeploymentsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewUpgradeClusterNodeDeploymentsUnauthorized creates a UpgradeClusterNodeDeploymentsUnauthorized with default headers values
func NewUpgradeClusterNodeDeploymentsUnauthorized() *UpgradeClusterNodeDeploymentsUnauthorized {
	return &UpgradeClusterNodeDeploymentsUnauthorized{}
}

/*UpgradeClusterNodeDeploymentsUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type UpgradeClusterNodeDeploymentsUnauthorized struct {
}

func (o *UpgradeClusterNodeDeploymentsUnauthorized) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/upgrades][%d] upgradeClusterNodeDeploymentsUnauthorized ", 401)
}

func (o *UpgradeClusterNodeDeploymentsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewUpgradeClusterNodeDeploymentsForbidden creates a UpgradeClusterNodeDeploymentsForbidden with default headers values
func NewUpgradeClusterNodeDeploymentsForbidden() *UpgradeClusterNodeDeploymentsForbidden {
	return &UpgradeClusterNodeDeploymentsForbidden{}
}

/*UpgradeClusterNodeDeploymentsForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type UpgradeClusterNodeDeploymentsForbidden struct {
}

func (o *UpgradeClusterNodeDeploymentsForbidden) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/upgrades][%d] upgradeClusterNodeDeploymentsForbidden ", 403)
}

func (o *UpgradeClusterNodeDeploymentsForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewUpgradeClusterNodeDeploymentsDefault creates a UpgradeClusterNodeDeploymentsDefault with default headers values
func NewUpgradeClusterNodeDeploymentsDefault(code int) *UpgradeClusterNodeDeploymentsDefault {
	return &UpgradeClusterNodeDeploymentsDefault{
		_statusCode: code,
	}
}

/*UpgradeClusterNodeDeploymentsDefault handles this case with default header values.

errorResponse
*/
type UpgradeClusterNodeDeploymentsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the upgrade cluster node deployments default response
func (o *UpgradeClusterNodeDeploymentsDefault) Code() int {
	return o._statusCode
}

func (o *UpgradeClusterNodeDeploymentsDefault) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/upgrades][%d] upgradeClusterNodeDeployments default  %+v", o._statusCode, o.Payload)
}

func (o *UpgradeClusterNodeDeploymentsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *UpgradeClusterNodeDeploymentsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
