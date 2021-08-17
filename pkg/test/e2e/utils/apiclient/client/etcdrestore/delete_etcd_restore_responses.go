// Code generated by go-swagger; DO NOT EDIT.

package etcdrestore

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// DeleteEtcdRestoreReader is a Reader for the DeleteEtcdRestore structure.
type DeleteEtcdRestoreReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeleteEtcdRestoreReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewDeleteEtcdRestoreOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewDeleteEtcdRestoreUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewDeleteEtcdRestoreForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 409:
		result := NewDeleteEtcdRestoreConflict()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewDeleteEtcdRestoreDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewDeleteEtcdRestoreOK creates a DeleteEtcdRestoreOK with default headers values
func NewDeleteEtcdRestoreOK() *DeleteEtcdRestoreOK {
	return &DeleteEtcdRestoreOK{}
}

/* DeleteEtcdRestoreOK describes a response with status code 200, with default header values.

EmptyResponse is a empty response
*/
type DeleteEtcdRestoreOK struct {
}

func (o *DeleteEtcdRestoreOK) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/etcdrestores/{er_name}][%d] deleteEtcdRestoreOK ", 200)
}

func (o *DeleteEtcdRestoreOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteEtcdRestoreUnauthorized creates a DeleteEtcdRestoreUnauthorized with default headers values
func NewDeleteEtcdRestoreUnauthorized() *DeleteEtcdRestoreUnauthorized {
	return &DeleteEtcdRestoreUnauthorized{}
}

/* DeleteEtcdRestoreUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type DeleteEtcdRestoreUnauthorized struct {
}

func (o *DeleteEtcdRestoreUnauthorized) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/etcdrestores/{er_name}][%d] deleteEtcdRestoreUnauthorized ", 401)
}

func (o *DeleteEtcdRestoreUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteEtcdRestoreForbidden creates a DeleteEtcdRestoreForbidden with default headers values
func NewDeleteEtcdRestoreForbidden() *DeleteEtcdRestoreForbidden {
	return &DeleteEtcdRestoreForbidden{}
}

/* DeleteEtcdRestoreForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type DeleteEtcdRestoreForbidden struct {
}

func (o *DeleteEtcdRestoreForbidden) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/etcdrestores/{er_name}][%d] deleteEtcdRestoreForbidden ", 403)
}

func (o *DeleteEtcdRestoreForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteEtcdRestoreConflict creates a DeleteEtcdRestoreConflict with default headers values
func NewDeleteEtcdRestoreConflict() *DeleteEtcdRestoreConflict {
	return &DeleteEtcdRestoreConflict{}
}

/* DeleteEtcdRestoreConflict describes a response with status code 409, with default header values.

errorResponse
*/
type DeleteEtcdRestoreConflict struct {
	Payload *models.ErrorResponse
}

func (o *DeleteEtcdRestoreConflict) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/etcdrestores/{er_name}][%d] deleteEtcdRestoreConflict  %+v", 409, o.Payload)
}
func (o *DeleteEtcdRestoreConflict) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *DeleteEtcdRestoreConflict) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewDeleteEtcdRestoreDefault creates a DeleteEtcdRestoreDefault with default headers values
func NewDeleteEtcdRestoreDefault(code int) *DeleteEtcdRestoreDefault {
	return &DeleteEtcdRestoreDefault{
		_statusCode: code,
	}
}

/* DeleteEtcdRestoreDefault describes a response with status code -1, with default header values.

errorResponse
*/
type DeleteEtcdRestoreDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the delete etcd restore default response
func (o *DeleteEtcdRestoreDefault) Code() int {
	return o._statusCode
}

func (o *DeleteEtcdRestoreDefault) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/etcdrestores/{er_name}][%d] deleteEtcdRestore default  %+v", o._statusCode, o.Payload)
}
func (o *DeleteEtcdRestoreDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *DeleteEtcdRestoreDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
