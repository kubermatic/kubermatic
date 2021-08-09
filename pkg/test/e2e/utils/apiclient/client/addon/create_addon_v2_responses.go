// Code generated by go-swagger; DO NOT EDIT.

package addon

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// CreateAddonV2Reader is a Reader for the CreateAddonV2 structure.
type CreateAddonV2Reader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *CreateAddonV2Reader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 201:
		result := NewCreateAddonV2Created()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewCreateAddonV2Unauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewCreateAddonV2Forbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewCreateAddonV2Default(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewCreateAddonV2Created creates a CreateAddonV2Created with default headers values
func NewCreateAddonV2Created() *CreateAddonV2Created {
	return &CreateAddonV2Created{}
}

/* CreateAddonV2Created describes a response with status code 201, with default header values.

Addon
*/
type CreateAddonV2Created struct {
	Payload *models.Addon
}

func (o *CreateAddonV2Created) Error() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/addons][%d] createAddonV2Created  %+v", 201, o.Payload)
}
func (o *CreateAddonV2Created) GetPayload() *models.Addon {
	return o.Payload
}

func (o *CreateAddonV2Created) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Addon)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewCreateAddonV2Unauthorized creates a CreateAddonV2Unauthorized with default headers values
func NewCreateAddonV2Unauthorized() *CreateAddonV2Unauthorized {
	return &CreateAddonV2Unauthorized{}
}

/* CreateAddonV2Unauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type CreateAddonV2Unauthorized struct {
}

func (o *CreateAddonV2Unauthorized) Error() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/addons][%d] createAddonV2Unauthorized ", 401)
}

func (o *CreateAddonV2Unauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateAddonV2Forbidden creates a CreateAddonV2Forbidden with default headers values
func NewCreateAddonV2Forbidden() *CreateAddonV2Forbidden {
	return &CreateAddonV2Forbidden{}
}

/* CreateAddonV2Forbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type CreateAddonV2Forbidden struct {
}

func (o *CreateAddonV2Forbidden) Error() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/addons][%d] createAddonV2Forbidden ", 403)
}

func (o *CreateAddonV2Forbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateAddonV2Default creates a CreateAddonV2Default with default headers values
func NewCreateAddonV2Default(code int) *CreateAddonV2Default {
	return &CreateAddonV2Default{
		_statusCode: code,
	}
}

/* CreateAddonV2Default describes a response with status code -1, with default header values.

errorResponse
*/
type CreateAddonV2Default struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the create addon v2 default response
func (o *CreateAddonV2Default) Code() int {
	return o._statusCode
}

func (o *CreateAddonV2Default) Error() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/addons][%d] createAddonV2 default  %+v", o._statusCode, o.Payload)
}
func (o *CreateAddonV2Default) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *CreateAddonV2Default) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
