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

// GetClusterHealthReader is a Reader for the GetClusterHealth structure.
type GetClusterHealthReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetClusterHealthReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetClusterHealthOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewGetClusterHealthUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewGetClusterHealthForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewGetClusterHealthDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetClusterHealthOK creates a GetClusterHealthOK with default headers values
func NewGetClusterHealthOK() *GetClusterHealthOK {
	return &GetClusterHealthOK{}
}

/* GetClusterHealthOK describes a response with status code 200, with default header values.

ClusterHealth
*/
type GetClusterHealthOK struct {
	Payload *models.ClusterHealth
}

// IsSuccess returns true when this get cluster health o k response has a 2xx status code
func (o *GetClusterHealthOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this get cluster health o k response has a 3xx status code
func (o *GetClusterHealthOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get cluster health o k response has a 4xx status code
func (o *GetClusterHealthOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this get cluster health o k response has a 5xx status code
func (o *GetClusterHealthOK) IsServerError() bool {
	return false
}

// IsCode returns true when this get cluster health o k response a status code equal to that given
func (o *GetClusterHealthOK) IsCode(code int) bool {
	return code == 200
}

func (o *GetClusterHealthOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/health][%d] getClusterHealthOK  %+v", 200, o.Payload)
}

func (o *GetClusterHealthOK) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/health][%d] getClusterHealthOK  %+v", 200, o.Payload)
}

func (o *GetClusterHealthOK) GetPayload() *models.ClusterHealth {
	return o.Payload
}

func (o *GetClusterHealthOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ClusterHealth)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetClusterHealthUnauthorized creates a GetClusterHealthUnauthorized with default headers values
func NewGetClusterHealthUnauthorized() *GetClusterHealthUnauthorized {
	return &GetClusterHealthUnauthorized{}
}

/* GetClusterHealthUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type GetClusterHealthUnauthorized struct {
}

// IsSuccess returns true when this get cluster health unauthorized response has a 2xx status code
func (o *GetClusterHealthUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this get cluster health unauthorized response has a 3xx status code
func (o *GetClusterHealthUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get cluster health unauthorized response has a 4xx status code
func (o *GetClusterHealthUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this get cluster health unauthorized response has a 5xx status code
func (o *GetClusterHealthUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this get cluster health unauthorized response a status code equal to that given
func (o *GetClusterHealthUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *GetClusterHealthUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/health][%d] getClusterHealthUnauthorized ", 401)
}

func (o *GetClusterHealthUnauthorized) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/health][%d] getClusterHealthUnauthorized ", 401)
}

func (o *GetClusterHealthUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetClusterHealthForbidden creates a GetClusterHealthForbidden with default headers values
func NewGetClusterHealthForbidden() *GetClusterHealthForbidden {
	return &GetClusterHealthForbidden{}
}

/* GetClusterHealthForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type GetClusterHealthForbidden struct {
}

// IsSuccess returns true when this get cluster health forbidden response has a 2xx status code
func (o *GetClusterHealthForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this get cluster health forbidden response has a 3xx status code
func (o *GetClusterHealthForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get cluster health forbidden response has a 4xx status code
func (o *GetClusterHealthForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this get cluster health forbidden response has a 5xx status code
func (o *GetClusterHealthForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this get cluster health forbidden response a status code equal to that given
func (o *GetClusterHealthForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *GetClusterHealthForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/health][%d] getClusterHealthForbidden ", 403)
}

func (o *GetClusterHealthForbidden) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/health][%d] getClusterHealthForbidden ", 403)
}

func (o *GetClusterHealthForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetClusterHealthDefault creates a GetClusterHealthDefault with default headers values
func NewGetClusterHealthDefault(code int) *GetClusterHealthDefault {
	return &GetClusterHealthDefault{
		_statusCode: code,
	}
}

/* GetClusterHealthDefault describes a response with status code -1, with default header values.

errorResponse
*/
type GetClusterHealthDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the get cluster health default response
func (o *GetClusterHealthDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this get cluster health default response has a 2xx status code
func (o *GetClusterHealthDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this get cluster health default response has a 3xx status code
func (o *GetClusterHealthDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this get cluster health default response has a 4xx status code
func (o *GetClusterHealthDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this get cluster health default response has a 5xx status code
func (o *GetClusterHealthDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this get cluster health default response a status code equal to that given
func (o *GetClusterHealthDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *GetClusterHealthDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/health][%d] getClusterHealth default  %+v", o._statusCode, o.Payload)
}

func (o *GetClusterHealthDefault) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/health][%d] getClusterHealth default  %+v", o._statusCode, o.Payload)
}

func (o *GetClusterHealthDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *GetClusterHealthDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
