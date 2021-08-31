// Code generated by go-swagger; DO NOT EDIT.

package metering

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListMeteringReportsReader is a Reader for the ListMeteringReports structure.
type ListMeteringReportsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListMeteringReportsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListMeteringReportsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListMeteringReportsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListMeteringReportsForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListMeteringReportsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListMeteringReportsOK creates a ListMeteringReportsOK with default headers values
func NewListMeteringReportsOK() *ListMeteringReportsOK {
	return &ListMeteringReportsOK{}
}

/* ListMeteringReportsOK describes a response with status code 200, with default header values.

MeteringReport
*/
type ListMeteringReportsOK struct {
	Payload []*models.MeteringReport
}

func (o *ListMeteringReportsOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin/metering/reports][%d] listMeteringReportsOK  %+v", 200, o.Payload)
}
func (o *ListMeteringReportsOK) GetPayload() []*models.MeteringReport {
	return o.Payload
}

func (o *ListMeteringReportsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListMeteringReportsUnauthorized creates a ListMeteringReportsUnauthorized with default headers values
func NewListMeteringReportsUnauthorized() *ListMeteringReportsUnauthorized {
	return &ListMeteringReportsUnauthorized{}
}

/* ListMeteringReportsUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type ListMeteringReportsUnauthorized struct {
}

func (o *ListMeteringReportsUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin/metering/reports][%d] listMeteringReportsUnauthorized ", 401)
}

func (o *ListMeteringReportsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListMeteringReportsForbidden creates a ListMeteringReportsForbidden with default headers values
func NewListMeteringReportsForbidden() *ListMeteringReportsForbidden {
	return &ListMeteringReportsForbidden{}
}

/* ListMeteringReportsForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type ListMeteringReportsForbidden struct {
}

func (o *ListMeteringReportsForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin/metering/reports][%d] listMeteringReportsForbidden ", 403)
}

func (o *ListMeteringReportsForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListMeteringReportsDefault creates a ListMeteringReportsDefault with default headers values
func NewListMeteringReportsDefault(code int) *ListMeteringReportsDefault {
	return &ListMeteringReportsDefault{
		_statusCode: code,
	}
}

/* ListMeteringReportsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListMeteringReportsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list metering reports default response
func (o *ListMeteringReportsDefault) Code() int {
	return o._statusCode
}

func (o *ListMeteringReportsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin/metering/reports][%d] listMeteringReports default  %+v", o._statusCode, o.Payload)
}
func (o *ListMeteringReportsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListMeteringReportsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
