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

// DeleteSSHKeyReader is a Reader for the DeleteSSHKey structure.
type DeleteSSHKeyReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeleteSSHKeyReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewDeleteSSHKeyOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewDeleteSSHKeyUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewDeleteSSHKeyForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewDeleteSSHKeyDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewDeleteSSHKeyOK creates a DeleteSSHKeyOK with default headers values
func NewDeleteSSHKeyOK() *DeleteSSHKeyOK {
	return &DeleteSSHKeyOK{}
}

/*DeleteSSHKeyOK handles this case with default header values.

EmptyResponse is a empty response
*/
type DeleteSSHKeyOK struct {
}

func (o *DeleteSSHKeyOK) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/projects/{project_id}/sshkeys/{key_id}][%d] deleteSshKeyOK ", 200)
}

func (o *DeleteSSHKeyOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteSSHKeyUnauthorized creates a DeleteSSHKeyUnauthorized with default headers values
func NewDeleteSSHKeyUnauthorized() *DeleteSSHKeyUnauthorized {
	return &DeleteSSHKeyUnauthorized{}
}

/*DeleteSSHKeyUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type DeleteSSHKeyUnauthorized struct {
}

func (o *DeleteSSHKeyUnauthorized) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/projects/{project_id}/sshkeys/{key_id}][%d] deleteSshKeyUnauthorized ", 401)
}

func (o *DeleteSSHKeyUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteSSHKeyForbidden creates a DeleteSSHKeyForbidden with default headers values
func NewDeleteSSHKeyForbidden() *DeleteSSHKeyForbidden {
	return &DeleteSSHKeyForbidden{}
}

/*DeleteSSHKeyForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type DeleteSSHKeyForbidden struct {
}

func (o *DeleteSSHKeyForbidden) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/projects/{project_id}/sshkeys/{key_id}][%d] deleteSshKeyForbidden ", 403)
}

func (o *DeleteSSHKeyForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteSSHKeyDefault creates a DeleteSSHKeyDefault with default headers values
func NewDeleteSSHKeyDefault(code int) *DeleteSSHKeyDefault {
	return &DeleteSSHKeyDefault{
		_statusCode: code,
	}
}

/*DeleteSSHKeyDefault handles this case with default header values.

errorResponse
*/
type DeleteSSHKeyDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the delete SSH key default response
func (o *DeleteSSHKeyDefault) Code() int {
	return o._statusCode
}

func (o *DeleteSSHKeyDefault) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/projects/{project_id}/sshkeys/{key_id}][%d] deleteSSHKey default  %+v", o._statusCode, o.Payload)
}

func (o *DeleteSSHKeyDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *DeleteSSHKeyDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
