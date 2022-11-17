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

// GetMeteringReportReader is a Reader for the GetMeteringReport structure.
type GetMeteringReportReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetMeteringReportReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetMeteringReportOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewGetMeteringReportUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewGetMeteringReportForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewGetMeteringReportDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetMeteringReportOK creates a GetMeteringReportOK with default headers values
func NewGetMeteringReportOK() *GetMeteringReportOK {
	return &GetMeteringReportOK{}
}

/* GetMeteringReportOK describes a response with status code 200, with default header values.

MeteringReportURL
*/
type GetMeteringReportOK struct {
	Payload models.ReportURL
}

func (o *GetMeteringReportOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin/metering/reports/{report_name}][%d] getMeteringReportOK  %+v", 200, o.Payload)
}
func (o *GetMeteringReportOK) GetPayload() models.ReportURL {
	return o.Payload
}

func (o *GetMeteringReportOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetMeteringReportUnauthorized creates a GetMeteringReportUnauthorized with default headers values
func NewGetMeteringReportUnauthorized() *GetMeteringReportUnauthorized {
	return &GetMeteringReportUnauthorized{}
}

/* GetMeteringReportUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type GetMeteringReportUnauthorized struct {
}

func (o *GetMeteringReportUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin/metering/reports/{report_name}][%d] getMeteringReportUnauthorized ", 401)
}

func (o *GetMeteringReportUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetMeteringReportForbidden creates a GetMeteringReportForbidden with default headers values
func NewGetMeteringReportForbidden() *GetMeteringReportForbidden {
	return &GetMeteringReportForbidden{}
}

/* GetMeteringReportForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type GetMeteringReportForbidden struct {
}

func (o *GetMeteringReportForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin/metering/reports/{report_name}][%d] getMeteringReportForbidden ", 403)
}

func (o *GetMeteringReportForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetMeteringReportDefault creates a GetMeteringReportDefault with default headers values
func NewGetMeteringReportDefault(code int) *GetMeteringReportDefault {
	return &GetMeteringReportDefault{
		_statusCode: code,
	}
}

/* GetMeteringReportDefault describes a response with status code -1, with default header values.

errorResponse
*/
type GetMeteringReportDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the get metering report default response
func (o *GetMeteringReportDefault) Code() int {
	return o._statusCode
}

func (o *GetMeteringReportDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/admin/metering/reports/{report_name}][%d] getMeteringReport default  %+v", o._statusCode, o.Payload)
}
func (o *GetMeteringReportDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *GetMeteringReportDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}