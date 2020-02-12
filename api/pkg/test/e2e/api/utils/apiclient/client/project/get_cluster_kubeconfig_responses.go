// Code generated by go-swagger; DO NOT EDIT.

package project

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// GetClusterKubeconfigReader is a Reader for the GetClusterKubeconfig structure.
type GetClusterKubeconfigReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetClusterKubeconfigReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetClusterKubeconfigOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewGetClusterKubeconfigUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewGetClusterKubeconfigForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewGetClusterKubeconfigDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetClusterKubeconfigOK creates a GetClusterKubeconfigOK with default headers values
func NewGetClusterKubeconfigOK() *GetClusterKubeconfigOK {
	return &GetClusterKubeconfigOK{}
}

/*GetClusterKubeconfigOK handles this case with default header values.

Kubeconfig is a clusters kubeconfig
*/
type GetClusterKubeconfigOK struct {
	Payload *models.Config
}

func (o *GetClusterKubeconfigOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/kubeconfig][%d] getClusterKubeconfigOK  %+v", 200, o.Payload)
}

func (o *GetClusterKubeconfigOK) GetPayload() *models.Config {
	return o.Payload
}

func (o *GetClusterKubeconfigOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Config)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetClusterKubeconfigUnauthorized creates a GetClusterKubeconfigUnauthorized with default headers values
func NewGetClusterKubeconfigUnauthorized() *GetClusterKubeconfigUnauthorized {
	return &GetClusterKubeconfigUnauthorized{}
}

/*GetClusterKubeconfigUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type GetClusterKubeconfigUnauthorized struct {
}

func (o *GetClusterKubeconfigUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/kubeconfig][%d] getClusterKubeconfigUnauthorized ", 401)
}

func (o *GetClusterKubeconfigUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetClusterKubeconfigForbidden creates a GetClusterKubeconfigForbidden with default headers values
func NewGetClusterKubeconfigForbidden() *GetClusterKubeconfigForbidden {
	return &GetClusterKubeconfigForbidden{}
}

/*GetClusterKubeconfigForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type GetClusterKubeconfigForbidden struct {
}

func (o *GetClusterKubeconfigForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/kubeconfig][%d] getClusterKubeconfigForbidden ", 403)
}

func (o *GetClusterKubeconfigForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetClusterKubeconfigDefault creates a GetClusterKubeconfigDefault with default headers values
func NewGetClusterKubeconfigDefault(code int) *GetClusterKubeconfigDefault {
	return &GetClusterKubeconfigDefault{
		_statusCode: code,
	}
}

/*GetClusterKubeconfigDefault handles this case with default header values.

errorResponse
*/
type GetClusterKubeconfigDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the get cluster kubeconfig default response
func (o *GetClusterKubeconfigDefault) Code() int {
	return o._statusCode
}

func (o *GetClusterKubeconfigDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/kubeconfig][%d] getClusterKubeconfig default  %+v", o._statusCode, o.Payload)
}

func (o *GetClusterKubeconfigDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *GetClusterKubeconfigDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
