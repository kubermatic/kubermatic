// Code generated by go-swagger; DO NOT EDIT.

package metric

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// ListNodeDeploymentMetricsReader is a Reader for the ListNodeDeploymentMetrics structure.
type ListNodeDeploymentMetricsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListNodeDeploymentMetricsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListNodeDeploymentMetricsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListNodeDeploymentMetricsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListNodeDeploymentMetricsForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListNodeDeploymentMetricsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListNodeDeploymentMetricsOK creates a ListNodeDeploymentMetricsOK with default headers values
func NewListNodeDeploymentMetricsOK() *ListNodeDeploymentMetricsOK {
	return &ListNodeDeploymentMetricsOK{}
}

/*ListNodeDeploymentMetricsOK handles this case with default header values.

NodeMetric
*/
type ListNodeDeploymentMetricsOK struct {
	Payload []*models.NodeMetric
}

func (o *ListNodeDeploymentMetricsOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}/nodes/metrics][%d] listNodeDeploymentMetricsOK  %+v", 200, o.Payload)
}

func (o *ListNodeDeploymentMetricsOK) GetPayload() []*models.NodeMetric {
	return o.Payload
}

func (o *ListNodeDeploymentMetricsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListNodeDeploymentMetricsUnauthorized creates a ListNodeDeploymentMetricsUnauthorized with default headers values
func NewListNodeDeploymentMetricsUnauthorized() *ListNodeDeploymentMetricsUnauthorized {
	return &ListNodeDeploymentMetricsUnauthorized{}
}

/*ListNodeDeploymentMetricsUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type ListNodeDeploymentMetricsUnauthorized struct {
}

func (o *ListNodeDeploymentMetricsUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}/nodes/metrics][%d] listNodeDeploymentMetricsUnauthorized ", 401)
}

func (o *ListNodeDeploymentMetricsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListNodeDeploymentMetricsForbidden creates a ListNodeDeploymentMetricsForbidden with default headers values
func NewListNodeDeploymentMetricsForbidden() *ListNodeDeploymentMetricsForbidden {
	return &ListNodeDeploymentMetricsForbidden{}
}

/*ListNodeDeploymentMetricsForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type ListNodeDeploymentMetricsForbidden struct {
}

func (o *ListNodeDeploymentMetricsForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}/nodes/metrics][%d] listNodeDeploymentMetricsForbidden ", 403)
}

func (o *ListNodeDeploymentMetricsForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListNodeDeploymentMetricsDefault creates a ListNodeDeploymentMetricsDefault with default headers values
func NewListNodeDeploymentMetricsDefault(code int) *ListNodeDeploymentMetricsDefault {
	return &ListNodeDeploymentMetricsDefault{
		_statusCode: code,
	}
}

/*ListNodeDeploymentMetricsDefault handles this case with default header values.

errorResponse
*/
type ListNodeDeploymentMetricsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list node deployment metrics default response
func (o *ListNodeDeploymentMetricsDefault) Code() int {
	return o._statusCode
}

func (o *ListNodeDeploymentMetricsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}/nodes/metrics][%d] listNodeDeploymentMetrics default  %+v", o._statusCode, o.Payload)
}

func (o *ListNodeDeploymentMetricsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListNodeDeploymentMetricsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
