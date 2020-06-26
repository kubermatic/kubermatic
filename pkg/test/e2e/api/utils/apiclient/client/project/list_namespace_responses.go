// Code generated by go-swagger; DO NOT EDIT.

package project

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/models"
)

// ListNamespaceReader is a Reader for the ListNamespace structure.
type ListNamespaceReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListNamespaceReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListNamespaceOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListNamespaceUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListNamespaceForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListNamespaceDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListNamespaceOK creates a ListNamespaceOK with default headers values
func NewListNamespaceOK() *ListNamespaceOK {
	return &ListNamespaceOK{}
}

/*ListNamespaceOK handles this case with default header values.

Namespace
*/
type ListNamespaceOK struct {
	Payload []*models.Namespace
}

func (o *ListNamespaceOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/namespaces][%d] listNamespaceOK  %+v", 200, o.Payload)
}

func (o *ListNamespaceOK) GetPayload() []*models.Namespace {
	return o.Payload
}

func (o *ListNamespaceOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListNamespaceUnauthorized creates a ListNamespaceUnauthorized with default headers values
func NewListNamespaceUnauthorized() *ListNamespaceUnauthorized {
	return &ListNamespaceUnauthorized{}
}

/*ListNamespaceUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type ListNamespaceUnauthorized struct {
}

func (o *ListNamespaceUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/namespaces][%d] listNamespaceUnauthorized ", 401)
}

func (o *ListNamespaceUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListNamespaceForbidden creates a ListNamespaceForbidden with default headers values
func NewListNamespaceForbidden() *ListNamespaceForbidden {
	return &ListNamespaceForbidden{}
}

/*ListNamespaceForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type ListNamespaceForbidden struct {
}

func (o *ListNamespaceForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/namespaces][%d] listNamespaceForbidden ", 403)
}

func (o *ListNamespaceForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListNamespaceDefault creates a ListNamespaceDefault with default headers values
func NewListNamespaceDefault(code int) *ListNamespaceDefault {
	return &ListNamespaceDefault{
		_statusCode: code,
	}
}

/*ListNamespaceDefault handles this case with default header values.

errorResponse
*/
type ListNamespaceDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list namespace default response
func (o *ListNamespaceDefault) Code() int {
	return o._statusCode
}

func (o *ListNamespaceDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/namespaces][%d] listNamespace default  %+v", o._statusCode, o.Payload)
}

func (o *ListNamespaceDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListNamespaceDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
