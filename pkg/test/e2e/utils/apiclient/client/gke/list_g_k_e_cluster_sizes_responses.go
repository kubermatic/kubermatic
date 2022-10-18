// Code generated by go-swagger; DO NOT EDIT.

package gke

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListGKEClusterSizesReader is a Reader for the ListGKEClusterSizes structure.
type ListGKEClusterSizesReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListGKEClusterSizesReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListGKEClusterSizesOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListGKEClusterSizesUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListGKEClusterSizesForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListGKEClusterSizesDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListGKEClusterSizesOK creates a ListGKEClusterSizesOK with default headers values
func NewListGKEClusterSizesOK() *ListGKEClusterSizesOK {
	return &ListGKEClusterSizesOK{}
}

/*
ListGKEClusterSizesOK describes a response with status code 200, with default header values.

GCPMachineSizeList
*/
type ListGKEClusterSizesOK struct {
	Payload models.GCPMachineSizeList
}

// IsSuccess returns true when this list g k e cluster sizes o k response has a 2xx status code
func (o *ListGKEClusterSizesOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list g k e cluster sizes o k response has a 3xx status code
func (o *ListGKEClusterSizesOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list g k e cluster sizes o k response has a 4xx status code
func (o *ListGKEClusterSizesOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list g k e cluster sizes o k response has a 5xx status code
func (o *ListGKEClusterSizesOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list g k e cluster sizes o k response a status code equal to that given
func (o *ListGKEClusterSizesOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListGKEClusterSizesOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/gke/sizes][%d] listGKEClusterSizesOK  %+v", 200, o.Payload)
}

func (o *ListGKEClusterSizesOK) String() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/gke/sizes][%d] listGKEClusterSizesOK  %+v", 200, o.Payload)
}

func (o *ListGKEClusterSizesOK) GetPayload() models.GCPMachineSizeList {
	return o.Payload
}

func (o *ListGKEClusterSizesOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListGKEClusterSizesUnauthorized creates a ListGKEClusterSizesUnauthorized with default headers values
func NewListGKEClusterSizesUnauthorized() *ListGKEClusterSizesUnauthorized {
	return &ListGKEClusterSizesUnauthorized{}
}

/*
ListGKEClusterSizesUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type ListGKEClusterSizesUnauthorized struct {
}

// IsSuccess returns true when this list g k e cluster sizes unauthorized response has a 2xx status code
func (o *ListGKEClusterSizesUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list g k e cluster sizes unauthorized response has a 3xx status code
func (o *ListGKEClusterSizesUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list g k e cluster sizes unauthorized response has a 4xx status code
func (o *ListGKEClusterSizesUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this list g k e cluster sizes unauthorized response has a 5xx status code
func (o *ListGKEClusterSizesUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this list g k e cluster sizes unauthorized response a status code equal to that given
func (o *ListGKEClusterSizesUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *ListGKEClusterSizesUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/gke/sizes][%d] listGKEClusterSizesUnauthorized ", 401)
}

func (o *ListGKEClusterSizesUnauthorized) String() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/gke/sizes][%d] listGKEClusterSizesUnauthorized ", 401)
}

func (o *ListGKEClusterSizesUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListGKEClusterSizesForbidden creates a ListGKEClusterSizesForbidden with default headers values
func NewListGKEClusterSizesForbidden() *ListGKEClusterSizesForbidden {
	return &ListGKEClusterSizesForbidden{}
}

/*
ListGKEClusterSizesForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type ListGKEClusterSizesForbidden struct {
}

// IsSuccess returns true when this list g k e cluster sizes forbidden response has a 2xx status code
func (o *ListGKEClusterSizesForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list g k e cluster sizes forbidden response has a 3xx status code
func (o *ListGKEClusterSizesForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list g k e cluster sizes forbidden response has a 4xx status code
func (o *ListGKEClusterSizesForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this list g k e cluster sizes forbidden response has a 5xx status code
func (o *ListGKEClusterSizesForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this list g k e cluster sizes forbidden response a status code equal to that given
func (o *ListGKEClusterSizesForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *ListGKEClusterSizesForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/gke/sizes][%d] listGKEClusterSizesForbidden ", 403)
}

func (o *ListGKEClusterSizesForbidden) String() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/gke/sizes][%d] listGKEClusterSizesForbidden ", 403)
}

func (o *ListGKEClusterSizesForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListGKEClusterSizesDefault creates a ListGKEClusterSizesDefault with default headers values
func NewListGKEClusterSizesDefault(code int) *ListGKEClusterSizesDefault {
	return &ListGKEClusterSizesDefault{
		_statusCode: code,
	}
}

/*
ListGKEClusterSizesDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListGKEClusterSizesDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list g k e cluster sizes default response
func (o *ListGKEClusterSizesDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list g k e cluster sizes default response has a 2xx status code
func (o *ListGKEClusterSizesDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list g k e cluster sizes default response has a 3xx status code
func (o *ListGKEClusterSizesDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list g k e cluster sizes default response has a 4xx status code
func (o *ListGKEClusterSizesDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list g k e cluster sizes default response has a 5xx status code
func (o *ListGKEClusterSizesDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list g k e cluster sizes default response a status code equal to that given
func (o *ListGKEClusterSizesDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListGKEClusterSizesDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/gke/sizes][%d] listGKEClusterSizes default  %+v", o._statusCode, o.Payload)
}

func (o *ListGKEClusterSizesDefault) String() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/gke/sizes][%d] listGKEClusterSizes default  %+v", o._statusCode, o.Payload)
}

func (o *ListGKEClusterSizesDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListGKEClusterSizesDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
