// Code generated by go-swagger; DO NOT EDIT.

package admin

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"fmt"
	"io"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// CreateSeedReader is a Reader for the CreateSeed structure.
type CreateSeedReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *CreateSeedReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewCreateSeedOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewCreateSeedUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewCreateSeedForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewCreateSeedDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewCreateSeedOK creates a CreateSeedOK with default headers values
func NewCreateSeedOK() *CreateSeedOK {
	return &CreateSeedOK{}
}

/* CreateSeedOK describes a response with status code 200, with default header values.

Seed
*/
type CreateSeedOK struct {
	Payload *models.Seed
}

func (o *CreateSeedOK) Error() string {
	return fmt.Sprintf("[POST /api/v1/admin/seeds][%d] createSeedOK  %+v", 200, o.Payload)
}
func (o *CreateSeedOK) GetPayload() *models.Seed {
	return o.Payload
}

func (o *CreateSeedOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Seed)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewCreateSeedUnauthorized creates a CreateSeedUnauthorized with default headers values
func NewCreateSeedUnauthorized() *CreateSeedUnauthorized {
	return &CreateSeedUnauthorized{}
}

/* CreateSeedUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type CreateSeedUnauthorized struct {
}

func (o *CreateSeedUnauthorized) Error() string {
	return fmt.Sprintf("[POST /api/v1/admin/seeds][%d] createSeedUnauthorized ", 401)
}

func (o *CreateSeedUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateSeedForbidden creates a CreateSeedForbidden with default headers values
func NewCreateSeedForbidden() *CreateSeedForbidden {
	return &CreateSeedForbidden{}
}

/* CreateSeedForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type CreateSeedForbidden struct {
}

func (o *CreateSeedForbidden) Error() string {
	return fmt.Sprintf("[POST /api/v1/admin/seeds][%d] createSeedForbidden ", 403)
}

func (o *CreateSeedForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateSeedDefault creates a CreateSeedDefault with default headers values
func NewCreateSeedDefault(code int) *CreateSeedDefault {
	return &CreateSeedDefault{
		_statusCode: code,
	}
}

/* CreateSeedDefault describes a response with status code -1, with default header values.

errorResponse
*/
type CreateSeedDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the create seed default response
func (o *CreateSeedDefault) Code() int {
	return o._statusCode
}

func (o *CreateSeedDefault) Error() string {
	return fmt.Sprintf("[POST /api/v1/admin/seeds][%d] createSeed default  %+v", o._statusCode, o.Payload)
}
func (o *CreateSeedDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *CreateSeedDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

/*CreateSeedBody create seed body
swagger:model CreateSeedBody
*/
type CreateSeedBody struct {

	// name
	Name string `json:"name,omitempty"`

	// spec
	Spec *models.CreateSeedSpec `json:"spec,omitempty"`
}

// Validate validates this create seed body
func (o *CreateSeedBody) Validate(formats strfmt.Registry) error {
	var res []error

	if err := o.validateSpec(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *CreateSeedBody) validateSpec(formats strfmt.Registry) error {
	if swag.IsZero(o.Spec) { // not required
		return nil
	}

	if o.Spec != nil {
		if err := o.Spec.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("Body" + "." + "spec")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("Body" + "." + "spec")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this create seed body based on the context it is used
func (o *CreateSeedBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := o.contextValidateSpec(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *CreateSeedBody) contextValidateSpec(ctx context.Context, formats strfmt.Registry) error {

	if o.Spec != nil {
		if err := o.Spec.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("Body" + "." + "spec")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("Body" + "." + "spec")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (o *CreateSeedBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *CreateSeedBody) UnmarshalBinary(b []byte) error {
	var res CreateSeedBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}
