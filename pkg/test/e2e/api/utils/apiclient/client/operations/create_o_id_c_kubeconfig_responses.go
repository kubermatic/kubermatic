// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/api/utils/apiclient/models"
)

// CreateOIDCKubeconfigReader is a Reader for the CreateOIDCKubeconfig structure.
type CreateOIDCKubeconfigReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *CreateOIDCKubeconfigReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewCreateOIDCKubeconfigOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewCreateOIDCKubeconfigDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewCreateOIDCKubeconfigOK creates a CreateOIDCKubeconfigOK with default headers values
func NewCreateOIDCKubeconfigOK() *CreateOIDCKubeconfigOK {
	return &CreateOIDCKubeconfigOK{}
}

/*CreateOIDCKubeconfigOK handles this case with default header values.

Kubeconfig is a clusters kubeconfig
*/
type CreateOIDCKubeconfigOK struct {
	Payload []uint8
}

func (o *CreateOIDCKubeconfigOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/kubeconfig][%d] createOIdCKubeconfigOK  %+v", 200, o.Payload)
}

func (o *CreateOIDCKubeconfigOK) GetPayload() []uint8 {
	return o.Payload
}

func (o *CreateOIDCKubeconfigOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewCreateOIDCKubeconfigDefault creates a CreateOIDCKubeconfigDefault with default headers values
func NewCreateOIDCKubeconfigDefault(code int) *CreateOIDCKubeconfigDefault {
	return &CreateOIDCKubeconfigDefault{
		_statusCode: code,
	}
}

/*CreateOIDCKubeconfigDefault handles this case with default header values.

errorResponse
*/
type CreateOIDCKubeconfigDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the create o ID c kubeconfig default response
func (o *CreateOIDCKubeconfigDefault) Code() int {
	return o._statusCode
}

func (o *CreateOIDCKubeconfigDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/kubeconfig][%d] createOIDCKubeconfig default  %+v", o._statusCode, o.Payload)
}

func (o *CreateOIDCKubeconfigDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *CreateOIDCKubeconfigDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
