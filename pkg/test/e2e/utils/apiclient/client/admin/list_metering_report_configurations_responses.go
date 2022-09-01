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

// ListMeteringReportConfigurationsReader is a Reader for the ListMeteringReportConfigurations structure.
type ListMeteringReportConfigurationsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListMeteringReportConfigurationsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListMeteringReportConfigurationsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListMeteringReportConfigurationsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListMeteringReportConfigurationsForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListMeteringReportConfigurationsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListMeteringReportConfigurationsOK creates a ListMeteringReportConfigurationsOK with default headers values
func NewListMeteringReportConfigurationsOK() *ListMeteringReportConfigurationsOK {
	return &ListMeteringReportConfigurationsOK{}
}

/* ListMeteringReportConfigurationsOK describes a response with status code 200, with default header values.

MeteringReportConfiguration
*/
type ListMeteringReportConfigurationsOK struct {
	Payload []*models.MeteringReportConfiguration
}

// IsSuccess returns true when this list metering report configurations o k response has a 2xx status code
func (o *ListMeteringReportConfigurationsOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list metering report configurations o k response has a 3xx status code
func (o *ListMeteringReportConfigurationsOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list metering report configurations o k response has a 4xx status code
func (o *ListMeteringReportConfigurationsOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list metering report configurations o k response has a 5xx status code
func (o *ListMeteringReportConfigurationsOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list metering report configurations o k response a status code equal to that given
func (o *ListMeteringReportConfigurationsOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListMeteringReportConfigurationsOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin/metering/configurations/reports][%d] listMeteringReportConfigurationsOK  %+v", 200, o.Payload)
}

func (o *ListMeteringReportConfigurationsOK) String() string {
	return fmt.Sprintf("[GET /api/v1/admin/metering/configurations/reports][%d] listMeteringReportConfigurationsOK  %+v", 200, o.Payload)
}

func (o *ListMeteringReportConfigurationsOK) GetPayload() []*models.MeteringReportConfiguration {
	return o.Payload
}

func (o *ListMeteringReportConfigurationsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListMeteringReportConfigurationsUnauthorized creates a ListMeteringReportConfigurationsUnauthorized with default headers values
func NewListMeteringReportConfigurationsUnauthorized() *ListMeteringReportConfigurationsUnauthorized {
	return &ListMeteringReportConfigurationsUnauthorized{}
}

/* ListMeteringReportConfigurationsUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type ListMeteringReportConfigurationsUnauthorized struct {
}

// IsSuccess returns true when this list metering report configurations unauthorized response has a 2xx status code
func (o *ListMeteringReportConfigurationsUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list metering report configurations unauthorized response has a 3xx status code
func (o *ListMeteringReportConfigurationsUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list metering report configurations unauthorized response has a 4xx status code
func (o *ListMeteringReportConfigurationsUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this list metering report configurations unauthorized response has a 5xx status code
func (o *ListMeteringReportConfigurationsUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this list metering report configurations unauthorized response a status code equal to that given
func (o *ListMeteringReportConfigurationsUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *ListMeteringReportConfigurationsUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin/metering/configurations/reports][%d] listMeteringReportConfigurationsUnauthorized ", 401)
}

func (o *ListMeteringReportConfigurationsUnauthorized) String() string {
	return fmt.Sprintf("[GET /api/v1/admin/metering/configurations/reports][%d] listMeteringReportConfigurationsUnauthorized ", 401)
}

func (o *ListMeteringReportConfigurationsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListMeteringReportConfigurationsForbidden creates a ListMeteringReportConfigurationsForbidden with default headers values
func NewListMeteringReportConfigurationsForbidden() *ListMeteringReportConfigurationsForbidden {
	return &ListMeteringReportConfigurationsForbidden{}
}

/* ListMeteringReportConfigurationsForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type ListMeteringReportConfigurationsForbidden struct {
}

// IsSuccess returns true when this list metering report configurations forbidden response has a 2xx status code
func (o *ListMeteringReportConfigurationsForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list metering report configurations forbidden response has a 3xx status code
func (o *ListMeteringReportConfigurationsForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list metering report configurations forbidden response has a 4xx status code
func (o *ListMeteringReportConfigurationsForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this list metering report configurations forbidden response has a 5xx status code
func (o *ListMeteringReportConfigurationsForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this list metering report configurations forbidden response a status code equal to that given
func (o *ListMeteringReportConfigurationsForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *ListMeteringReportConfigurationsForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin/metering/configurations/reports][%d] listMeteringReportConfigurationsForbidden ", 403)
}

func (o *ListMeteringReportConfigurationsForbidden) String() string {
	return fmt.Sprintf("[GET /api/v1/admin/metering/configurations/reports][%d] listMeteringReportConfigurationsForbidden ", 403)
}

func (o *ListMeteringReportConfigurationsForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListMeteringReportConfigurationsDefault creates a ListMeteringReportConfigurationsDefault with default headers values
func NewListMeteringReportConfigurationsDefault(code int) *ListMeteringReportConfigurationsDefault {
	return &ListMeteringReportConfigurationsDefault{
		_statusCode: code,
	}
}

/* ListMeteringReportConfigurationsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListMeteringReportConfigurationsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list metering report configurations default response
func (o *ListMeteringReportConfigurationsDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list metering report configurations default response has a 2xx status code
func (o *ListMeteringReportConfigurationsDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list metering report configurations default response has a 3xx status code
func (o *ListMeteringReportConfigurationsDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list metering report configurations default response has a 4xx status code
func (o *ListMeteringReportConfigurationsDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list metering report configurations default response has a 5xx status code
func (o *ListMeteringReportConfigurationsDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list metering report configurations default response a status code equal to that given
func (o *ListMeteringReportConfigurationsDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListMeteringReportConfigurationsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin/metering/configurations/reports][%d] listMeteringReportConfigurations default  %+v", o._statusCode, o.Payload)
}

func (o *ListMeteringReportConfigurationsDefault) String() string {
	return fmt.Sprintf("[GET /api/v1/admin/metering/configurations/reports][%d] listMeteringReportConfigurations default  %+v", o._statusCode, o.Payload)
}

func (o *ListMeteringReportConfigurationsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListMeteringReportConfigurationsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
