// Code generated by go-swagger; DO NOT EDIT.

package eks

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListProjectEKSVersionsReader is a Reader for the ListProjectEKSVersions structure.
type ListProjectEKSVersionsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListProjectEKSVersionsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListProjectEKSVersionsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListProjectEKSVersionsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListProjectEKSVersionsOK creates a ListProjectEKSVersionsOK with default headers values
func NewListProjectEKSVersionsOK() *ListProjectEKSVersionsOK {
	return &ListProjectEKSVersionsOK{}
}

/*
ListProjectEKSVersionsOK describes a response with status code 200, with default header values.

MasterVersion
*/
type ListProjectEKSVersionsOK struct {
	Payload []*models.MasterVersion
}

// IsSuccess returns true when this list project e k s versions o k response has a 2xx status code
func (o *ListProjectEKSVersionsOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list project e k s versions o k response has a 3xx status code
func (o *ListProjectEKSVersionsOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list project e k s versions o k response has a 4xx status code
func (o *ListProjectEKSVersionsOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list project e k s versions o k response has a 5xx status code
func (o *ListProjectEKSVersionsOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list project e k s versions o k response a status code equal to that given
func (o *ListProjectEKSVersionsOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListProjectEKSVersionsOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/providers/eks/versions][%d] listProjectEKSVersionsOK  %+v", 200, o.Payload)
}

func (o *ListProjectEKSVersionsOK) String() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/providers/eks/versions][%d] listProjectEKSVersionsOK  %+v", 200, o.Payload)
}

func (o *ListProjectEKSVersionsOK) GetPayload() []*models.MasterVersion {
	return o.Payload
}

func (o *ListProjectEKSVersionsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListProjectEKSVersionsDefault creates a ListProjectEKSVersionsDefault with default headers values
func NewListProjectEKSVersionsDefault(code int) *ListProjectEKSVersionsDefault {
	return &ListProjectEKSVersionsDefault{
		_statusCode: code,
	}
}

/*
ListProjectEKSVersionsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListProjectEKSVersionsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list project e k s versions default response
func (o *ListProjectEKSVersionsDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list project e k s versions default response has a 2xx status code
func (o *ListProjectEKSVersionsDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list project e k s versions default response has a 3xx status code
func (o *ListProjectEKSVersionsDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list project e k s versions default response has a 4xx status code
func (o *ListProjectEKSVersionsDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list project e k s versions default response has a 5xx status code
func (o *ListProjectEKSVersionsDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list project e k s versions default response a status code equal to that given
func (o *ListProjectEKSVersionsDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListProjectEKSVersionsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/providers/eks/versions][%d] listProjectEKSVersions default  %+v", o._statusCode, o.Payload)
}

func (o *ListProjectEKSVersionsDefault) String() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/providers/eks/versions][%d] listProjectEKSVersions default  %+v", o._statusCode, o.Payload)
}

func (o *ListProjectEKSVersionsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListProjectEKSVersionsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
