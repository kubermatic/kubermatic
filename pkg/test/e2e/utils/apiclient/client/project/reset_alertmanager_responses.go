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

// ResetAlertmanagerReader is a Reader for the ResetAlertmanager structure.
type ResetAlertmanagerReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ResetAlertmanagerReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewResetAlertmanagerOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewResetAlertmanagerUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewResetAlertmanagerForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewResetAlertmanagerDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewResetAlertmanagerOK creates a ResetAlertmanagerOK with default headers values
func NewResetAlertmanagerOK() *ResetAlertmanagerOK {
	return &ResetAlertmanagerOK{}
}

/*
ResetAlertmanagerOK describes a response with status code 200, with default header values.

EmptyResponse is a empty response
*/
type ResetAlertmanagerOK struct {
}

// IsSuccess returns true when this reset alertmanager o k response has a 2xx status code
func (o *ResetAlertmanagerOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this reset alertmanager o k response has a 3xx status code
func (o *ResetAlertmanagerOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this reset alertmanager o k response has a 4xx status code
func (o *ResetAlertmanagerOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this reset alertmanager o k response has a 5xx status code
func (o *ResetAlertmanagerOK) IsServerError() bool {
	return false
}

// IsCode returns true when this reset alertmanager o k response a status code equal to that given
func (o *ResetAlertmanagerOK) IsCode(code int) bool {
	return code == 200
}

func (o *ResetAlertmanagerOK) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/alertmanager/config][%d] resetAlertmanagerOK ", 200)
}

func (o *ResetAlertmanagerOK) String() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/alertmanager/config][%d] resetAlertmanagerOK ", 200)
}

func (o *ResetAlertmanagerOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewResetAlertmanagerUnauthorized creates a ResetAlertmanagerUnauthorized with default headers values
func NewResetAlertmanagerUnauthorized() *ResetAlertmanagerUnauthorized {
	return &ResetAlertmanagerUnauthorized{}
}

/*
ResetAlertmanagerUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type ResetAlertmanagerUnauthorized struct {
}

// IsSuccess returns true when this reset alertmanager unauthorized response has a 2xx status code
func (o *ResetAlertmanagerUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this reset alertmanager unauthorized response has a 3xx status code
func (o *ResetAlertmanagerUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this reset alertmanager unauthorized response has a 4xx status code
func (o *ResetAlertmanagerUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this reset alertmanager unauthorized response has a 5xx status code
func (o *ResetAlertmanagerUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this reset alertmanager unauthorized response a status code equal to that given
func (o *ResetAlertmanagerUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *ResetAlertmanagerUnauthorized) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/alertmanager/config][%d] resetAlertmanagerUnauthorized ", 401)
}

func (o *ResetAlertmanagerUnauthorized) String() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/alertmanager/config][%d] resetAlertmanagerUnauthorized ", 401)
}

func (o *ResetAlertmanagerUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewResetAlertmanagerForbidden creates a ResetAlertmanagerForbidden with default headers values
func NewResetAlertmanagerForbidden() *ResetAlertmanagerForbidden {
	return &ResetAlertmanagerForbidden{}
}

/*
ResetAlertmanagerForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type ResetAlertmanagerForbidden struct {
}

// IsSuccess returns true when this reset alertmanager forbidden response has a 2xx status code
func (o *ResetAlertmanagerForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this reset alertmanager forbidden response has a 3xx status code
func (o *ResetAlertmanagerForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this reset alertmanager forbidden response has a 4xx status code
func (o *ResetAlertmanagerForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this reset alertmanager forbidden response has a 5xx status code
func (o *ResetAlertmanagerForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this reset alertmanager forbidden response a status code equal to that given
func (o *ResetAlertmanagerForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *ResetAlertmanagerForbidden) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/alertmanager/config][%d] resetAlertmanagerForbidden ", 403)
}

func (o *ResetAlertmanagerForbidden) String() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/alertmanager/config][%d] resetAlertmanagerForbidden ", 403)
}

func (o *ResetAlertmanagerForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewResetAlertmanagerDefault creates a ResetAlertmanagerDefault with default headers values
func NewResetAlertmanagerDefault(code int) *ResetAlertmanagerDefault {
	return &ResetAlertmanagerDefault{
		_statusCode: code,
	}
}

/*
ResetAlertmanagerDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ResetAlertmanagerDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the reset alertmanager default response
func (o *ResetAlertmanagerDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this reset alertmanager default response has a 2xx status code
func (o *ResetAlertmanagerDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this reset alertmanager default response has a 3xx status code
func (o *ResetAlertmanagerDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this reset alertmanager default response has a 4xx status code
func (o *ResetAlertmanagerDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this reset alertmanager default response has a 5xx status code
func (o *ResetAlertmanagerDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this reset alertmanager default response a status code equal to that given
func (o *ResetAlertmanagerDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ResetAlertmanagerDefault) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/alertmanager/config][%d] resetAlertmanager default  %+v", o._statusCode, o.Payload)
}

func (o *ResetAlertmanagerDefault) String() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/alertmanager/config][%d] resetAlertmanager default  %+v", o._statusCode, o.Payload)
}

func (o *ResetAlertmanagerDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ResetAlertmanagerDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
