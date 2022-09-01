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

// GetClusterEventsReader is a Reader for the GetClusterEvents structure.
type GetClusterEventsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetClusterEventsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetClusterEventsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewGetClusterEventsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewGetClusterEventsForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewGetClusterEventsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetClusterEventsOK creates a GetClusterEventsOK with default headers values
func NewGetClusterEventsOK() *GetClusterEventsOK {
	return &GetClusterEventsOK{}
}

/* GetClusterEventsOK describes a response with status code 200, with default header values.

Event
*/
type GetClusterEventsOK struct {
	Payload []*models.Event
}

// IsSuccess returns true when this get cluster events o k response has a 2xx status code
func (o *GetClusterEventsOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this get cluster events o k response has a 3xx status code
func (o *GetClusterEventsOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get cluster events o k response has a 4xx status code
func (o *GetClusterEventsOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this get cluster events o k response has a 5xx status code
func (o *GetClusterEventsOK) IsServerError() bool {
	return false
}

// IsCode returns true when this get cluster events o k response a status code equal to that given
func (o *GetClusterEventsOK) IsCode(code int) bool {
	return code == 200
}

func (o *GetClusterEventsOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/events][%d] getClusterEventsOK  %+v", 200, o.Payload)
}

func (o *GetClusterEventsOK) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/events][%d] getClusterEventsOK  %+v", 200, o.Payload)
}

func (o *GetClusterEventsOK) GetPayload() []*models.Event {
	return o.Payload
}

func (o *GetClusterEventsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetClusterEventsUnauthorized creates a GetClusterEventsUnauthorized with default headers values
func NewGetClusterEventsUnauthorized() *GetClusterEventsUnauthorized {
	return &GetClusterEventsUnauthorized{}
}

/* GetClusterEventsUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type GetClusterEventsUnauthorized struct {
}

// IsSuccess returns true when this get cluster events unauthorized response has a 2xx status code
func (o *GetClusterEventsUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this get cluster events unauthorized response has a 3xx status code
func (o *GetClusterEventsUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get cluster events unauthorized response has a 4xx status code
func (o *GetClusterEventsUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this get cluster events unauthorized response has a 5xx status code
func (o *GetClusterEventsUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this get cluster events unauthorized response a status code equal to that given
func (o *GetClusterEventsUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *GetClusterEventsUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/events][%d] getClusterEventsUnauthorized ", 401)
}

func (o *GetClusterEventsUnauthorized) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/events][%d] getClusterEventsUnauthorized ", 401)
}

func (o *GetClusterEventsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetClusterEventsForbidden creates a GetClusterEventsForbidden with default headers values
func NewGetClusterEventsForbidden() *GetClusterEventsForbidden {
	return &GetClusterEventsForbidden{}
}

/* GetClusterEventsForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type GetClusterEventsForbidden struct {
}

// IsSuccess returns true when this get cluster events forbidden response has a 2xx status code
func (o *GetClusterEventsForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this get cluster events forbidden response has a 3xx status code
func (o *GetClusterEventsForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get cluster events forbidden response has a 4xx status code
func (o *GetClusterEventsForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this get cluster events forbidden response has a 5xx status code
func (o *GetClusterEventsForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this get cluster events forbidden response a status code equal to that given
func (o *GetClusterEventsForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *GetClusterEventsForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/events][%d] getClusterEventsForbidden ", 403)
}

func (o *GetClusterEventsForbidden) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/events][%d] getClusterEventsForbidden ", 403)
}

func (o *GetClusterEventsForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetClusterEventsDefault creates a GetClusterEventsDefault with default headers values
func NewGetClusterEventsDefault(code int) *GetClusterEventsDefault {
	return &GetClusterEventsDefault{
		_statusCode: code,
	}
}

/* GetClusterEventsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type GetClusterEventsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the get cluster events default response
func (o *GetClusterEventsDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this get cluster events default response has a 2xx status code
func (o *GetClusterEventsDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this get cluster events default response has a 3xx status code
func (o *GetClusterEventsDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this get cluster events default response has a 4xx status code
func (o *GetClusterEventsDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this get cluster events default response has a 5xx status code
func (o *GetClusterEventsDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this get cluster events default response a status code equal to that given
func (o *GetClusterEventsDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *GetClusterEventsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/events][%d] getClusterEvents default  %+v", o._statusCode, o.Payload)
}

func (o *GetClusterEventsDefault) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/events][%d] getClusterEvents default  %+v", o._statusCode, o.Payload)
}

func (o *GetClusterEventsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *GetClusterEventsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
