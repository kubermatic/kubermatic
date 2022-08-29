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

// GetClusterMetricsReader is a Reader for the GetClusterMetrics structure.
type GetClusterMetricsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetClusterMetricsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetClusterMetricsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewGetClusterMetricsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewGetClusterMetricsForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewGetClusterMetricsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetClusterMetricsOK creates a GetClusterMetricsOK with default headers values
func NewGetClusterMetricsOK() *GetClusterMetricsOK {
	return &GetClusterMetricsOK{}
}

/*
GetClusterMetricsOK describes a response with status code 200, with default header values.

ClusterMetrics
*/
type GetClusterMetricsOK struct {
	Payload *models.ClusterMetrics
}

// IsSuccess returns true when this get cluster metrics o k response has a 2xx status code
func (o *GetClusterMetricsOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this get cluster metrics o k response has a 3xx status code
func (o *GetClusterMetricsOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get cluster metrics o k response has a 4xx status code
func (o *GetClusterMetricsOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this get cluster metrics o k response has a 5xx status code
func (o *GetClusterMetricsOK) IsServerError() bool {
	return false
}

// IsCode returns true when this get cluster metrics o k response a status code equal to that given
func (o *GetClusterMetricsOK) IsCode(code int) bool {
	return code == 200
}

func (o *GetClusterMetricsOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/metrics][%d] getClusterMetricsOK  %+v", 200, o.Payload)
}

func (o *GetClusterMetricsOK) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/metrics][%d] getClusterMetricsOK  %+v", 200, o.Payload)
}

func (o *GetClusterMetricsOK) GetPayload() *models.ClusterMetrics {
	return o.Payload
}

func (o *GetClusterMetricsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ClusterMetrics)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetClusterMetricsUnauthorized creates a GetClusterMetricsUnauthorized with default headers values
func NewGetClusterMetricsUnauthorized() *GetClusterMetricsUnauthorized {
	return &GetClusterMetricsUnauthorized{}
}

/*
GetClusterMetricsUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type GetClusterMetricsUnauthorized struct {
}

// IsSuccess returns true when this get cluster metrics unauthorized response has a 2xx status code
func (o *GetClusterMetricsUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this get cluster metrics unauthorized response has a 3xx status code
func (o *GetClusterMetricsUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get cluster metrics unauthorized response has a 4xx status code
func (o *GetClusterMetricsUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this get cluster metrics unauthorized response has a 5xx status code
func (o *GetClusterMetricsUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this get cluster metrics unauthorized response a status code equal to that given
func (o *GetClusterMetricsUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *GetClusterMetricsUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/metrics][%d] getClusterMetricsUnauthorized ", 401)
}

func (o *GetClusterMetricsUnauthorized) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/metrics][%d] getClusterMetricsUnauthorized ", 401)
}

func (o *GetClusterMetricsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetClusterMetricsForbidden creates a GetClusterMetricsForbidden with default headers values
func NewGetClusterMetricsForbidden() *GetClusterMetricsForbidden {
	return &GetClusterMetricsForbidden{}
}

/*
GetClusterMetricsForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type GetClusterMetricsForbidden struct {
}

// IsSuccess returns true when this get cluster metrics forbidden response has a 2xx status code
func (o *GetClusterMetricsForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this get cluster metrics forbidden response has a 3xx status code
func (o *GetClusterMetricsForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get cluster metrics forbidden response has a 4xx status code
func (o *GetClusterMetricsForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this get cluster metrics forbidden response has a 5xx status code
func (o *GetClusterMetricsForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this get cluster metrics forbidden response a status code equal to that given
func (o *GetClusterMetricsForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *GetClusterMetricsForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/metrics][%d] getClusterMetricsForbidden ", 403)
}

func (o *GetClusterMetricsForbidden) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/metrics][%d] getClusterMetricsForbidden ", 403)
}

func (o *GetClusterMetricsForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetClusterMetricsDefault creates a GetClusterMetricsDefault with default headers values
func NewGetClusterMetricsDefault(code int) *GetClusterMetricsDefault {
	return &GetClusterMetricsDefault{
		_statusCode: code,
	}
}

/*
GetClusterMetricsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type GetClusterMetricsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the get cluster metrics default response
func (o *GetClusterMetricsDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this get cluster metrics default response has a 2xx status code
func (o *GetClusterMetricsDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this get cluster metrics default response has a 3xx status code
func (o *GetClusterMetricsDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this get cluster metrics default response has a 4xx status code
func (o *GetClusterMetricsDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this get cluster metrics default response has a 5xx status code
func (o *GetClusterMetricsDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this get cluster metrics default response a status code equal to that given
func (o *GetClusterMetricsDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *GetClusterMetricsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/metrics][%d] getClusterMetrics default  %+v", o._statusCode, o.Payload)
}

func (o *GetClusterMetricsDefault) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/metrics][%d] getClusterMetrics default  %+v", o._statusCode, o.Payload)
}

func (o *GetClusterMetricsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *GetClusterMetricsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
