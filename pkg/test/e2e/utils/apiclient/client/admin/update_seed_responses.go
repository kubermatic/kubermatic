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

// UpdateSeedReader is a Reader for the UpdateSeed structure.
type UpdateSeedReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *UpdateSeedReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewUpdateSeedOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewUpdateSeedUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewUpdateSeedForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewUpdateSeedDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewUpdateSeedOK creates a UpdateSeedOK with default headers values
func NewUpdateSeedOK() *UpdateSeedOK {
	return &UpdateSeedOK{}
}

/* UpdateSeedOK describes a response with status code 200, with default header values.

Seed
*/
type UpdateSeedOK struct {
	Payload *models.Seed
}

func (o *UpdateSeedOK) Error() string {
	return fmt.Sprintf("[PATCH /api/v1/admin/seeds/{seed_name}][%d] updateSeedOK  %+v", 200, o.Payload)
}
func (o *UpdateSeedOK) GetPayload() *models.Seed {
	return o.Payload
}

func (o *UpdateSeedOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Seed)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewUpdateSeedUnauthorized creates a UpdateSeedUnauthorized with default headers values
func NewUpdateSeedUnauthorized() *UpdateSeedUnauthorized {
	return &UpdateSeedUnauthorized{}
}

/* UpdateSeedUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type UpdateSeedUnauthorized struct {
}

func (o *UpdateSeedUnauthorized) Error() string {
	return fmt.Sprintf("[PATCH /api/v1/admin/seeds/{seed_name}][%d] updateSeedUnauthorized ", 401)
}

func (o *UpdateSeedUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewUpdateSeedForbidden creates a UpdateSeedForbidden with default headers values
func NewUpdateSeedForbidden() *UpdateSeedForbidden {
	return &UpdateSeedForbidden{}
}

/* UpdateSeedForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type UpdateSeedForbidden struct {
}

func (o *UpdateSeedForbidden) Error() string {
	return fmt.Sprintf("[PATCH /api/v1/admin/seeds/{seed_name}][%d] updateSeedForbidden ", 403)
}

func (o *UpdateSeedForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewUpdateSeedDefault creates a UpdateSeedDefault with default headers values
func NewUpdateSeedDefault(code int) *UpdateSeedDefault {
	return &UpdateSeedDefault{
		_statusCode: code,
	}
}

/* UpdateSeedDefault describes a response with status code -1, with default header values.

errorResponse
*/
type UpdateSeedDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the update seed default response
func (o *UpdateSeedDefault) Code() int {
	return o._statusCode
}

func (o *UpdateSeedDefault) Error() string {
	return fmt.Sprintf("[PATCH /api/v1/admin/seeds/{seed_name}][%d] updateSeed default  %+v", o._statusCode, o.Payload)
}
func (o *UpdateSeedDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *UpdateSeedDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

/*UpdateSeedBody update seed body
swagger:model UpdateSeedBody
*/
type UpdateSeedBody struct {

	// name
	Name string `json:"name,omitempty"`

	// RawKubeconfig raw kubeconfig decoded to base64
	RawKubeconfig string `json:"raw_kubeconfig,omitempty"`

	// spec
	Spec *models.SeedSpec `json:"spec,omitempty"`
}

// Validate validates this update seed body
func (o *UpdateSeedBody) Validate(formats strfmt.Registry) error {
	var res []error

	if err := o.validateSpec(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *UpdateSeedBody) validateSpec(formats strfmt.Registry) error {
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

// ContextValidate validate this update seed body based on the context it is used
func (o *UpdateSeedBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := o.contextValidateSpec(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *UpdateSeedBody) contextValidateSpec(ctx context.Context, formats strfmt.Registry) error {

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
func (o *UpdateSeedBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *UpdateSeedBody) UnmarshalBinary(b []byte) error {
	var res UpdateSeedBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}