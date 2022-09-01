// Code generated by go-swagger; DO NOT EDIT.

package admin

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// UpdateMeteringReportConfigurationReader is a Reader for the UpdateMeteringReportConfiguration structure.
type UpdateMeteringReportConfigurationReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *UpdateMeteringReportConfigurationReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewUpdateMeteringReportConfigurationOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewUpdateMeteringReportConfigurationUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewUpdateMeteringReportConfigurationForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewUpdateMeteringReportConfigurationDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewUpdateMeteringReportConfigurationOK creates a UpdateMeteringReportConfigurationOK with default headers values
func NewUpdateMeteringReportConfigurationOK() *UpdateMeteringReportConfigurationOK {
	return &UpdateMeteringReportConfigurationOK{}
}

/* UpdateMeteringReportConfigurationOK describes a response with status code 200, with default header values.

MeteringReportConfiguration
*/
type UpdateMeteringReportConfigurationOK struct {
	Payload *models.MeteringReportConfiguration
}

// IsSuccess returns true when this update metering report configuration o k response has a 2xx status code
func (o *UpdateMeteringReportConfigurationOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this update metering report configuration o k response has a 3xx status code
func (o *UpdateMeteringReportConfigurationOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this update metering report configuration o k response has a 4xx status code
func (o *UpdateMeteringReportConfigurationOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this update metering report configuration o k response has a 5xx status code
func (o *UpdateMeteringReportConfigurationOK) IsServerError() bool {
	return false
}

// IsCode returns true when this update metering report configuration o k response a status code equal to that given
func (o *UpdateMeteringReportConfigurationOK) IsCode(code int) bool {
	return code == 200
}

func (o *UpdateMeteringReportConfigurationOK) Error() string {
	return fmt.Sprintf("[PUT /api/v1/admin/metering/configurations/reports/{name}][%d] updateMeteringReportConfigurationOK  %+v", 200, o.Payload)
}

func (o *UpdateMeteringReportConfigurationOK) String() string {
	return fmt.Sprintf("[PUT /api/v1/admin/metering/configurations/reports/{name}][%d] updateMeteringReportConfigurationOK  %+v", 200, o.Payload)
}

func (o *UpdateMeteringReportConfigurationOK) GetPayload() *models.MeteringReportConfiguration {
	return o.Payload
}

func (o *UpdateMeteringReportConfigurationOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.MeteringReportConfiguration)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewUpdateMeteringReportConfigurationUnauthorized creates a UpdateMeteringReportConfigurationUnauthorized with default headers values
func NewUpdateMeteringReportConfigurationUnauthorized() *UpdateMeteringReportConfigurationUnauthorized {
	return &UpdateMeteringReportConfigurationUnauthorized{}
}

/* UpdateMeteringReportConfigurationUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type UpdateMeteringReportConfigurationUnauthorized struct {
}

// IsSuccess returns true when this update metering report configuration unauthorized response has a 2xx status code
func (o *UpdateMeteringReportConfigurationUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this update metering report configuration unauthorized response has a 3xx status code
func (o *UpdateMeteringReportConfigurationUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this update metering report configuration unauthorized response has a 4xx status code
func (o *UpdateMeteringReportConfigurationUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this update metering report configuration unauthorized response has a 5xx status code
func (o *UpdateMeteringReportConfigurationUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this update metering report configuration unauthorized response a status code equal to that given
func (o *UpdateMeteringReportConfigurationUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *UpdateMeteringReportConfigurationUnauthorized) Error() string {
	return fmt.Sprintf("[PUT /api/v1/admin/metering/configurations/reports/{name}][%d] updateMeteringReportConfigurationUnauthorized ", 401)
}

func (o *UpdateMeteringReportConfigurationUnauthorized) String() string {
	return fmt.Sprintf("[PUT /api/v1/admin/metering/configurations/reports/{name}][%d] updateMeteringReportConfigurationUnauthorized ", 401)
}

func (o *UpdateMeteringReportConfigurationUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewUpdateMeteringReportConfigurationForbidden creates a UpdateMeteringReportConfigurationForbidden with default headers values
func NewUpdateMeteringReportConfigurationForbidden() *UpdateMeteringReportConfigurationForbidden {
	return &UpdateMeteringReportConfigurationForbidden{}
}

/* UpdateMeteringReportConfigurationForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type UpdateMeteringReportConfigurationForbidden struct {
}

// IsSuccess returns true when this update metering report configuration forbidden response has a 2xx status code
func (o *UpdateMeteringReportConfigurationForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this update metering report configuration forbidden response has a 3xx status code
func (o *UpdateMeteringReportConfigurationForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this update metering report configuration forbidden response has a 4xx status code
func (o *UpdateMeteringReportConfigurationForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this update metering report configuration forbidden response has a 5xx status code
func (o *UpdateMeteringReportConfigurationForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this update metering report configuration forbidden response a status code equal to that given
func (o *UpdateMeteringReportConfigurationForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *UpdateMeteringReportConfigurationForbidden) Error() string {
	return fmt.Sprintf("[PUT /api/v1/admin/metering/configurations/reports/{name}][%d] updateMeteringReportConfigurationForbidden ", 403)
}

func (o *UpdateMeteringReportConfigurationForbidden) String() string {
	return fmt.Sprintf("[PUT /api/v1/admin/metering/configurations/reports/{name}][%d] updateMeteringReportConfigurationForbidden ", 403)
}

func (o *UpdateMeteringReportConfigurationForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewUpdateMeteringReportConfigurationDefault creates a UpdateMeteringReportConfigurationDefault with default headers values
func NewUpdateMeteringReportConfigurationDefault(code int) *UpdateMeteringReportConfigurationDefault {
	return &UpdateMeteringReportConfigurationDefault{
		_statusCode: code,
	}
}

/* UpdateMeteringReportConfigurationDefault describes a response with status code -1, with default header values.

errorResponse
*/
type UpdateMeteringReportConfigurationDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the update metering report configuration default response
func (o *UpdateMeteringReportConfigurationDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this update metering report configuration default response has a 2xx status code
func (o *UpdateMeteringReportConfigurationDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this update metering report configuration default response has a 3xx status code
func (o *UpdateMeteringReportConfigurationDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this update metering report configuration default response has a 4xx status code
func (o *UpdateMeteringReportConfigurationDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this update metering report configuration default response has a 5xx status code
func (o *UpdateMeteringReportConfigurationDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this update metering report configuration default response a status code equal to that given
func (o *UpdateMeteringReportConfigurationDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *UpdateMeteringReportConfigurationDefault) Error() string {
	return fmt.Sprintf("[PUT /api/v1/admin/metering/configurations/reports/{name}][%d] updateMeteringReportConfiguration default  %+v", o._statusCode, o.Payload)
}

func (o *UpdateMeteringReportConfigurationDefault) String() string {
	return fmt.Sprintf("[PUT /api/v1/admin/metering/configurations/reports/{name}][%d] updateMeteringReportConfiguration default  %+v", o._statusCode, o.Payload)
}

func (o *UpdateMeteringReportConfigurationDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *UpdateMeteringReportConfigurationDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

/*UpdateMeteringReportConfigurationBody update metering report configuration body
swagger:model UpdateMeteringReportConfigurationBody
*/
type UpdateMeteringReportConfigurationBody struct {

	// interval
	Interval int32 `json:"interval,omitempty"`

	// retention
	Retention int32 `json:"retention,omitempty"`

	// schedule
	Schedule string `json:"schedule,omitempty"`

	// types
	Types []string `json:"types"`
}

// Validate validates this update metering report configuration body
func (o *UpdateMeteringReportConfigurationBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this update metering report configuration body based on context it is used
func (o *UpdateMeteringReportConfigurationBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *UpdateMeteringReportConfigurationBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *UpdateMeteringReportConfigurationBody) UnmarshalBinary(b []byte) error {
	var res UpdateMeteringReportConfigurationBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}
