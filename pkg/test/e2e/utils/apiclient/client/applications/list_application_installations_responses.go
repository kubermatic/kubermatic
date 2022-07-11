// Code generated by go-swagger; DO NOT EDIT.

package applications

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListApplicationInstallationsReader is a Reader for the ListApplicationInstallations structure.
type ListApplicationInstallationsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListApplicationInstallationsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListApplicationInstallationsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListApplicationInstallationsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListApplicationInstallationsForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListApplicationInstallationsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListApplicationInstallationsOK creates a ListApplicationInstallationsOK with default headers values
func NewListApplicationInstallationsOK() *ListApplicationInstallationsOK {
	return &ListApplicationInstallationsOK{}
}

/* ListApplicationInstallationsOK describes a response with status code 200, with default header values.

ApplicationInstallation
*/
type ListApplicationInstallationsOK struct {
	Payload []*models.ApplicationInstallation
}

func (o *ListApplicationInstallationsOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/applicationinstallations][%d] listApplicationInstallationsOK  %+v", 200, o.Payload)
}
func (o *ListApplicationInstallationsOK) GetPayload() []*models.ApplicationInstallation {
	return o.Payload
}

func (o *ListApplicationInstallationsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListApplicationInstallationsUnauthorized creates a ListApplicationInstallationsUnauthorized with default headers values
func NewListApplicationInstallationsUnauthorized() *ListApplicationInstallationsUnauthorized {
	return &ListApplicationInstallationsUnauthorized{}
}

/* ListApplicationInstallationsUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type ListApplicationInstallationsUnauthorized struct {
}

func (o *ListApplicationInstallationsUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/applicationinstallations][%d] listApplicationInstallationsUnauthorized ", 401)
}

func (o *ListApplicationInstallationsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListApplicationInstallationsForbidden creates a ListApplicationInstallationsForbidden with default headers values
func NewListApplicationInstallationsForbidden() *ListApplicationInstallationsForbidden {
	return &ListApplicationInstallationsForbidden{}
}

/* ListApplicationInstallationsForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type ListApplicationInstallationsForbidden struct {
}

func (o *ListApplicationInstallationsForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/applicationinstallations][%d] listApplicationInstallationsForbidden ", 403)
}

func (o *ListApplicationInstallationsForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListApplicationInstallationsDefault creates a ListApplicationInstallationsDefault with default headers values
func NewListApplicationInstallationsDefault(code int) *ListApplicationInstallationsDefault {
	return &ListApplicationInstallationsDefault{
		_statusCode: code,
	}
}

/* ListApplicationInstallationsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListApplicationInstallationsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list application installations default response
func (o *ListApplicationInstallationsDefault) Code() int {
	return o._statusCode
}

func (o *ListApplicationInstallationsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/applicationinstallations][%d] listApplicationInstallations default  %+v", o._statusCode, o.Payload)
}
func (o *ListApplicationInstallationsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListApplicationInstallationsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
