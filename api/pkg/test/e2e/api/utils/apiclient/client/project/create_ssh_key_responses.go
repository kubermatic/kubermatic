// Code generated by go-swagger; DO NOT EDIT.

package project

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// CreateSSHKeyReader is a Reader for the CreateSSHKey structure.
type CreateSSHKeyReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *CreateSSHKeyReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewCreateSSHKeyOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewCreateSSHKeyUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewCreateSSHKeyForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewCreateSSHKeyDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewCreateSSHKeyOK creates a CreateSSHKeyOK with default headers values
func NewCreateSSHKeyOK() *CreateSSHKeyOK {
	return &CreateSSHKeyOK{}
}

/*CreateSSHKeyOK handles this case with default header values.

SSHKey
*/
type CreateSSHKeyOK struct {
	Payload *models.SSHKey
}

func (o *CreateSSHKeyOK) Error() string {
	return fmt.Sprintf("[POST /api/v1/projects/{project_id}/sshkeys][%d] createSshKeyOK  %+v", 200, o.Payload)
}

func (o *CreateSSHKeyOK) GetPayload() *models.SSHKey {
	return o.Payload
}

func (o *CreateSSHKeyOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.SSHKey)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewCreateSSHKeyUnauthorized creates a CreateSSHKeyUnauthorized with default headers values
func NewCreateSSHKeyUnauthorized() *CreateSSHKeyUnauthorized {
	return &CreateSSHKeyUnauthorized{}
}

/*CreateSSHKeyUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type CreateSSHKeyUnauthorized struct {
}

func (o *CreateSSHKeyUnauthorized) Error() string {
	return fmt.Sprintf("[POST /api/v1/projects/{project_id}/sshkeys][%d] createSshKeyUnauthorized ", 401)
}

func (o *CreateSSHKeyUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateSSHKeyForbidden creates a CreateSSHKeyForbidden with default headers values
func NewCreateSSHKeyForbidden() *CreateSSHKeyForbidden {
	return &CreateSSHKeyForbidden{}
}

/*CreateSSHKeyForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type CreateSSHKeyForbidden struct {
}

func (o *CreateSSHKeyForbidden) Error() string {
	return fmt.Sprintf("[POST /api/v1/projects/{project_id}/sshkeys][%d] createSshKeyForbidden ", 403)
}

func (o *CreateSSHKeyForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateSSHKeyDefault creates a CreateSSHKeyDefault with default headers values
func NewCreateSSHKeyDefault(code int) *CreateSSHKeyDefault {
	return &CreateSSHKeyDefault{
		_statusCode: code,
	}
}

/*CreateSSHKeyDefault handles this case with default header values.

errorResponse
*/
type CreateSSHKeyDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the create SSH key default response
func (o *CreateSSHKeyDefault) Code() int {
	return o._statusCode
}

func (o *CreateSSHKeyDefault) Error() string {
	return fmt.Sprintf("[POST /api/v1/projects/{project_id}/sshkeys][%d] createSSHKey default  %+v", o._statusCode, o.Payload)
}

func (o *CreateSSHKeyDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *CreateSSHKeyDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
