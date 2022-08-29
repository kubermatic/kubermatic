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

// GetOidcClusterKubeconfigReader is a Reader for the GetOidcClusterKubeconfig structure.
type GetOidcClusterKubeconfigReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetOidcClusterKubeconfigReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetOidcClusterKubeconfigOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewGetOidcClusterKubeconfigUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewGetOidcClusterKubeconfigForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewGetOidcClusterKubeconfigDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetOidcClusterKubeconfigOK creates a GetOidcClusterKubeconfigOK with default headers values
func NewGetOidcClusterKubeconfigOK() *GetOidcClusterKubeconfigOK {
	return &GetOidcClusterKubeconfigOK{}
}

/*
GetOidcClusterKubeconfigOK describes a response with status code 200, with default header values.

Kubeconfig is a clusters kubeconfig
*/
type GetOidcClusterKubeconfigOK struct {
	Payload []uint8
}

// IsSuccess returns true when this get oidc cluster kubeconfig o k response has a 2xx status code
func (o *GetOidcClusterKubeconfigOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this get oidc cluster kubeconfig o k response has a 3xx status code
func (o *GetOidcClusterKubeconfigOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get oidc cluster kubeconfig o k response has a 4xx status code
func (o *GetOidcClusterKubeconfigOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this get oidc cluster kubeconfig o k response has a 5xx status code
func (o *GetOidcClusterKubeconfigOK) IsServerError() bool {
	return false
}

// IsCode returns true when this get oidc cluster kubeconfig o k response a status code equal to that given
func (o *GetOidcClusterKubeconfigOK) IsCode(code int) bool {
	return code == 200
}

func (o *GetOidcClusterKubeconfigOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/oidckubeconfig][%d] getOidcClusterKubeconfigOK  %+v", 200, o.Payload)
}

func (o *GetOidcClusterKubeconfigOK) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/oidckubeconfig][%d] getOidcClusterKubeconfigOK  %+v", 200, o.Payload)
}

func (o *GetOidcClusterKubeconfigOK) GetPayload() []uint8 {
	return o.Payload
}

func (o *GetOidcClusterKubeconfigOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetOidcClusterKubeconfigUnauthorized creates a GetOidcClusterKubeconfigUnauthorized with default headers values
func NewGetOidcClusterKubeconfigUnauthorized() *GetOidcClusterKubeconfigUnauthorized {
	return &GetOidcClusterKubeconfigUnauthorized{}
}

/*
GetOidcClusterKubeconfigUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type GetOidcClusterKubeconfigUnauthorized struct {
}

// IsSuccess returns true when this get oidc cluster kubeconfig unauthorized response has a 2xx status code
func (o *GetOidcClusterKubeconfigUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this get oidc cluster kubeconfig unauthorized response has a 3xx status code
func (o *GetOidcClusterKubeconfigUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get oidc cluster kubeconfig unauthorized response has a 4xx status code
func (o *GetOidcClusterKubeconfigUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this get oidc cluster kubeconfig unauthorized response has a 5xx status code
func (o *GetOidcClusterKubeconfigUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this get oidc cluster kubeconfig unauthorized response a status code equal to that given
func (o *GetOidcClusterKubeconfigUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *GetOidcClusterKubeconfigUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/oidckubeconfig][%d] getOidcClusterKubeconfigUnauthorized ", 401)
}

func (o *GetOidcClusterKubeconfigUnauthorized) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/oidckubeconfig][%d] getOidcClusterKubeconfigUnauthorized ", 401)
}

func (o *GetOidcClusterKubeconfigUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetOidcClusterKubeconfigForbidden creates a GetOidcClusterKubeconfigForbidden with default headers values
func NewGetOidcClusterKubeconfigForbidden() *GetOidcClusterKubeconfigForbidden {
	return &GetOidcClusterKubeconfigForbidden{}
}

/*
GetOidcClusterKubeconfigForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type GetOidcClusterKubeconfigForbidden struct {
}

// IsSuccess returns true when this get oidc cluster kubeconfig forbidden response has a 2xx status code
func (o *GetOidcClusterKubeconfigForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this get oidc cluster kubeconfig forbidden response has a 3xx status code
func (o *GetOidcClusterKubeconfigForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get oidc cluster kubeconfig forbidden response has a 4xx status code
func (o *GetOidcClusterKubeconfigForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this get oidc cluster kubeconfig forbidden response has a 5xx status code
func (o *GetOidcClusterKubeconfigForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this get oidc cluster kubeconfig forbidden response a status code equal to that given
func (o *GetOidcClusterKubeconfigForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *GetOidcClusterKubeconfigForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/oidckubeconfig][%d] getOidcClusterKubeconfigForbidden ", 403)
}

func (o *GetOidcClusterKubeconfigForbidden) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/oidckubeconfig][%d] getOidcClusterKubeconfigForbidden ", 403)
}

func (o *GetOidcClusterKubeconfigForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetOidcClusterKubeconfigDefault creates a GetOidcClusterKubeconfigDefault with default headers values
func NewGetOidcClusterKubeconfigDefault(code int) *GetOidcClusterKubeconfigDefault {
	return &GetOidcClusterKubeconfigDefault{
		_statusCode: code,
	}
}

/*
GetOidcClusterKubeconfigDefault describes a response with status code -1, with default header values.

errorResponse
*/
type GetOidcClusterKubeconfigDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the get oidc cluster kubeconfig default response
func (o *GetOidcClusterKubeconfigDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this get oidc cluster kubeconfig default response has a 2xx status code
func (o *GetOidcClusterKubeconfigDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this get oidc cluster kubeconfig default response has a 3xx status code
func (o *GetOidcClusterKubeconfigDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this get oidc cluster kubeconfig default response has a 4xx status code
func (o *GetOidcClusterKubeconfigDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this get oidc cluster kubeconfig default response has a 5xx status code
func (o *GetOidcClusterKubeconfigDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this get oidc cluster kubeconfig default response a status code equal to that given
func (o *GetOidcClusterKubeconfigDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *GetOidcClusterKubeconfigDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/oidckubeconfig][%d] getOidcClusterKubeconfig default  %+v", o._statusCode, o.Payload)
}

func (o *GetOidcClusterKubeconfigDefault) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/oidckubeconfig][%d] getOidcClusterKubeconfig default  %+v", o._statusCode, o.Payload)
}

func (o *GetOidcClusterKubeconfigDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *GetOidcClusterKubeconfigDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
