// Code generated by go-swagger; DO NOT EDIT.

package gcp

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListGCPDiskTypesNoCredentialsV2Reader is a Reader for the ListGCPDiskTypesNoCredentialsV2 structure.
type ListGCPDiskTypesNoCredentialsV2Reader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListGCPDiskTypesNoCredentialsV2Reader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListGCPDiskTypesNoCredentialsV2OK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListGCPDiskTypesNoCredentialsV2Default(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListGCPDiskTypesNoCredentialsV2OK creates a ListGCPDiskTypesNoCredentialsV2OK with default headers values
func NewListGCPDiskTypesNoCredentialsV2OK() *ListGCPDiskTypesNoCredentialsV2OK {
	return &ListGCPDiskTypesNoCredentialsV2OK{}
}

/*
ListGCPDiskTypesNoCredentialsV2OK describes a response with status code 200, with default header values.

GCPDiskTypeList
*/
type ListGCPDiskTypesNoCredentialsV2OK struct {
	Payload models.GCPDiskTypeList
}

// IsSuccess returns true when this list g c p disk types no credentials v2 o k response has a 2xx status code
func (o *ListGCPDiskTypesNoCredentialsV2OK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list g c p disk types no credentials v2 o k response has a 3xx status code
func (o *ListGCPDiskTypesNoCredentialsV2OK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list g c p disk types no credentials v2 o k response has a 4xx status code
func (o *ListGCPDiskTypesNoCredentialsV2OK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list g c p disk types no credentials v2 o k response has a 5xx status code
func (o *ListGCPDiskTypesNoCredentialsV2OK) IsServerError() bool {
	return false
}

// IsCode returns true when this list g c p disk types no credentials v2 o k response a status code equal to that given
func (o *ListGCPDiskTypesNoCredentialsV2OK) IsCode(code int) bool {
	return code == 200
}

func (o *ListGCPDiskTypesNoCredentialsV2OK) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/gcp/disktypes][%d] listGCPDiskTypesNoCredentialsV2OK  %+v", 200, o.Payload)
}

func (o *ListGCPDiskTypesNoCredentialsV2OK) String() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/gcp/disktypes][%d] listGCPDiskTypesNoCredentialsV2OK  %+v", 200, o.Payload)
}

func (o *ListGCPDiskTypesNoCredentialsV2OK) GetPayload() models.GCPDiskTypeList {
	return o.Payload
}

func (o *ListGCPDiskTypesNoCredentialsV2OK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListGCPDiskTypesNoCredentialsV2Default creates a ListGCPDiskTypesNoCredentialsV2Default with default headers values
func NewListGCPDiskTypesNoCredentialsV2Default(code int) *ListGCPDiskTypesNoCredentialsV2Default {
	return &ListGCPDiskTypesNoCredentialsV2Default{
		_statusCode: code,
	}
}

/*
ListGCPDiskTypesNoCredentialsV2Default describes a response with status code -1, with default header values.

errorResponse
*/
type ListGCPDiskTypesNoCredentialsV2Default struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list g c p disk types no credentials v2 default response
func (o *ListGCPDiskTypesNoCredentialsV2Default) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list g c p disk types no credentials v2 default response has a 2xx status code
func (o *ListGCPDiskTypesNoCredentialsV2Default) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list g c p disk types no credentials v2 default response has a 3xx status code
func (o *ListGCPDiskTypesNoCredentialsV2Default) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list g c p disk types no credentials v2 default response has a 4xx status code
func (o *ListGCPDiskTypesNoCredentialsV2Default) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list g c p disk types no credentials v2 default response has a 5xx status code
func (o *ListGCPDiskTypesNoCredentialsV2Default) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list g c p disk types no credentials v2 default response a status code equal to that given
func (o *ListGCPDiskTypesNoCredentialsV2Default) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListGCPDiskTypesNoCredentialsV2Default) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/gcp/disktypes][%d] listGCPDiskTypesNoCredentialsV2 default  %+v", o._statusCode, o.Payload)
}

func (o *ListGCPDiskTypesNoCredentialsV2Default) String() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/gcp/disktypes][%d] listGCPDiskTypesNoCredentialsV2 default  %+v", o._statusCode, o.Payload)
}

func (o *ListGCPDiskTypesNoCredentialsV2Default) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListGCPDiskTypesNoCredentialsV2Default) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
