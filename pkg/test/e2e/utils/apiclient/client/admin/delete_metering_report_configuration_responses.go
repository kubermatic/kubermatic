// Code generated by go-swagger; DO NOT EDIT.

package admin

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// DeleteMeteringReportConfigurationReader is a Reader for the DeleteMeteringReportConfiguration structure.
type DeleteMeteringReportConfigurationReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeleteMeteringReportConfigurationReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewDeleteMeteringReportConfigurationOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewDeleteMeteringReportConfigurationUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewDeleteMeteringReportConfigurationForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewDeleteMeteringReportConfigurationDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewDeleteMeteringReportConfigurationOK creates a DeleteMeteringReportConfigurationOK with default headers values
func NewDeleteMeteringReportConfigurationOK() *DeleteMeteringReportConfigurationOK {
	return &DeleteMeteringReportConfigurationOK{}
}

/* DeleteMeteringReportConfigurationOK describes a response with status code 200, with default header values.

EmptyResponse is a empty response
*/
type DeleteMeteringReportConfigurationOK struct {
}

// IsSuccess returns true when this delete metering report configuration o k response has a 2xx status code
func (o *DeleteMeteringReportConfigurationOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this delete metering report configuration o k response has a 3xx status code
func (o *DeleteMeteringReportConfigurationOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this delete metering report configuration o k response has a 4xx status code
func (o *DeleteMeteringReportConfigurationOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this delete metering report configuration o k response has a 5xx status code
func (o *DeleteMeteringReportConfigurationOK) IsServerError() bool {
	return false
}

// IsCode returns true when this delete metering report configuration o k response a status code equal to that given
func (o *DeleteMeteringReportConfigurationOK) IsCode(code int) bool {
	return code == 200
}

func (o *DeleteMeteringReportConfigurationOK) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/admin/metering/configurations/reports/{name}][%d] deleteMeteringReportConfigurationOK ", 200)
}

func (o *DeleteMeteringReportConfigurationOK) String() string {
	return fmt.Sprintf("[DELETE /api/v1/admin/metering/configurations/reports/{name}][%d] deleteMeteringReportConfigurationOK ", 200)
}

func (o *DeleteMeteringReportConfigurationOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteMeteringReportConfigurationUnauthorized creates a DeleteMeteringReportConfigurationUnauthorized with default headers values
func NewDeleteMeteringReportConfigurationUnauthorized() *DeleteMeteringReportConfigurationUnauthorized {
	return &DeleteMeteringReportConfigurationUnauthorized{}
}

/* DeleteMeteringReportConfigurationUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type DeleteMeteringReportConfigurationUnauthorized struct {
}

// IsSuccess returns true when this delete metering report configuration unauthorized response has a 2xx status code
func (o *DeleteMeteringReportConfigurationUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this delete metering report configuration unauthorized response has a 3xx status code
func (o *DeleteMeteringReportConfigurationUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this delete metering report configuration unauthorized response has a 4xx status code
func (o *DeleteMeteringReportConfigurationUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this delete metering report configuration unauthorized response has a 5xx status code
func (o *DeleteMeteringReportConfigurationUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this delete metering report configuration unauthorized response a status code equal to that given
func (o *DeleteMeteringReportConfigurationUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *DeleteMeteringReportConfigurationUnauthorized) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/admin/metering/configurations/reports/{name}][%d] deleteMeteringReportConfigurationUnauthorized ", 401)
}

func (o *DeleteMeteringReportConfigurationUnauthorized) String() string {
	return fmt.Sprintf("[DELETE /api/v1/admin/metering/configurations/reports/{name}][%d] deleteMeteringReportConfigurationUnauthorized ", 401)
}

func (o *DeleteMeteringReportConfigurationUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteMeteringReportConfigurationForbidden creates a DeleteMeteringReportConfigurationForbidden with default headers values
func NewDeleteMeteringReportConfigurationForbidden() *DeleteMeteringReportConfigurationForbidden {
	return &DeleteMeteringReportConfigurationForbidden{}
}

/* DeleteMeteringReportConfigurationForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type DeleteMeteringReportConfigurationForbidden struct {
}

// IsSuccess returns true when this delete metering report configuration forbidden response has a 2xx status code
func (o *DeleteMeteringReportConfigurationForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this delete metering report configuration forbidden response has a 3xx status code
func (o *DeleteMeteringReportConfigurationForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this delete metering report configuration forbidden response has a 4xx status code
func (o *DeleteMeteringReportConfigurationForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this delete metering report configuration forbidden response has a 5xx status code
func (o *DeleteMeteringReportConfigurationForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this delete metering report configuration forbidden response a status code equal to that given
func (o *DeleteMeteringReportConfigurationForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *DeleteMeteringReportConfigurationForbidden) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/admin/metering/configurations/reports/{name}][%d] deleteMeteringReportConfigurationForbidden ", 403)
}

func (o *DeleteMeteringReportConfigurationForbidden) String() string {
	return fmt.Sprintf("[DELETE /api/v1/admin/metering/configurations/reports/{name}][%d] deleteMeteringReportConfigurationForbidden ", 403)
}

func (o *DeleteMeteringReportConfigurationForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteMeteringReportConfigurationDefault creates a DeleteMeteringReportConfigurationDefault with default headers values
func NewDeleteMeteringReportConfigurationDefault(code int) *DeleteMeteringReportConfigurationDefault {
	return &DeleteMeteringReportConfigurationDefault{
		_statusCode: code,
	}
}

/* DeleteMeteringReportConfigurationDefault describes a response with status code -1, with default header values.

errorResponse
*/
type DeleteMeteringReportConfigurationDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the delete metering report configuration default response
func (o *DeleteMeteringReportConfigurationDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this delete metering report configuration default response has a 2xx status code
func (o *DeleteMeteringReportConfigurationDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this delete metering report configuration default response has a 3xx status code
func (o *DeleteMeteringReportConfigurationDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this delete metering report configuration default response has a 4xx status code
func (o *DeleteMeteringReportConfigurationDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this delete metering report configuration default response has a 5xx status code
func (o *DeleteMeteringReportConfigurationDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this delete metering report configuration default response a status code equal to that given
func (o *DeleteMeteringReportConfigurationDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *DeleteMeteringReportConfigurationDefault) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/admin/metering/configurations/reports/{name}][%d] deleteMeteringReportConfiguration default  %+v", o._statusCode, o.Payload)
}

func (o *DeleteMeteringReportConfigurationDefault) String() string {
	return fmt.Sprintf("[DELETE /api/v1/admin/metering/configurations/reports/{name}][%d] deleteMeteringReportConfiguration default  %+v", o._statusCode, o.Payload)
}

func (o *DeleteMeteringReportConfigurationDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *DeleteMeteringReportConfigurationDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
