// Code generated by go-swagger; DO NOT EDIT.

package azure

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListAzureAvailabilityZonesNoCredentialsReader is a Reader for the ListAzureAvailabilityZonesNoCredentials structure.
type ListAzureAvailabilityZonesNoCredentialsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListAzureAvailabilityZonesNoCredentialsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListAzureAvailabilityZonesNoCredentialsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListAzureAvailabilityZonesNoCredentialsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListAzureAvailabilityZonesNoCredentialsOK creates a ListAzureAvailabilityZonesNoCredentialsOK with default headers values
func NewListAzureAvailabilityZonesNoCredentialsOK() *ListAzureAvailabilityZonesNoCredentialsOK {
	return &ListAzureAvailabilityZonesNoCredentialsOK{}
}

/* ListAzureAvailabilityZonesNoCredentialsOK describes a response with status code 200, with default header values.

AzureAvailabilityZonesList
*/
type ListAzureAvailabilityZonesNoCredentialsOK struct {
	Payload *models.AzureAvailabilityZonesList
}

// IsSuccess returns true when this list azure availability zones no credentials o k response has a 2xx status code
func (o *ListAzureAvailabilityZonesNoCredentialsOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list azure availability zones no credentials o k response has a 3xx status code
func (o *ListAzureAvailabilityZonesNoCredentialsOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list azure availability zones no credentials o k response has a 4xx status code
func (o *ListAzureAvailabilityZonesNoCredentialsOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list azure availability zones no credentials o k response has a 5xx status code
func (o *ListAzureAvailabilityZonesNoCredentialsOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list azure availability zones no credentials o k response a status code equal to that given
func (o *ListAzureAvailabilityZonesNoCredentialsOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListAzureAvailabilityZonesNoCredentialsOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/azure/availabilityzones][%d] listAzureAvailabilityZonesNoCredentialsOK  %+v", 200, o.Payload)
}

func (o *ListAzureAvailabilityZonesNoCredentialsOK) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/azure/availabilityzones][%d] listAzureAvailabilityZonesNoCredentialsOK  %+v", 200, o.Payload)
}

func (o *ListAzureAvailabilityZonesNoCredentialsOK) GetPayload() *models.AzureAvailabilityZonesList {
	return o.Payload
}

func (o *ListAzureAvailabilityZonesNoCredentialsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.AzureAvailabilityZonesList)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListAzureAvailabilityZonesNoCredentialsDefault creates a ListAzureAvailabilityZonesNoCredentialsDefault with default headers values
func NewListAzureAvailabilityZonesNoCredentialsDefault(code int) *ListAzureAvailabilityZonesNoCredentialsDefault {
	return &ListAzureAvailabilityZonesNoCredentialsDefault{
		_statusCode: code,
	}
}

/* ListAzureAvailabilityZonesNoCredentialsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListAzureAvailabilityZonesNoCredentialsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list azure availability zones no credentials default response
func (o *ListAzureAvailabilityZonesNoCredentialsDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list azure availability zones no credentials default response has a 2xx status code
func (o *ListAzureAvailabilityZonesNoCredentialsDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list azure availability zones no credentials default response has a 3xx status code
func (o *ListAzureAvailabilityZonesNoCredentialsDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list azure availability zones no credentials default response has a 4xx status code
func (o *ListAzureAvailabilityZonesNoCredentialsDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list azure availability zones no credentials default response has a 5xx status code
func (o *ListAzureAvailabilityZonesNoCredentialsDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list azure availability zones no credentials default response a status code equal to that given
func (o *ListAzureAvailabilityZonesNoCredentialsDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListAzureAvailabilityZonesNoCredentialsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/azure/availabilityzones][%d] listAzureAvailabilityZonesNoCredentials default  %+v", o._statusCode, o.Payload)
}

func (o *ListAzureAvailabilityZonesNoCredentialsDefault) String() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/azure/availabilityzones][%d] listAzureAvailabilityZonesNoCredentials default  %+v", o._statusCode, o.Payload)
}

func (o *ListAzureAvailabilityZonesNoCredentialsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListAzureAvailabilityZonesNoCredentialsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
