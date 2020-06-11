// Code generated by go-swagger; DO NOT EDIT.

package gcp

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/models"
)

// ListGCPDiskTypesNoCredentialsReader is a Reader for the ListGCPDiskTypesNoCredentials structure.
type ListGCPDiskTypesNoCredentialsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListGCPDiskTypesNoCredentialsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListGCPDiskTypesNoCredentialsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListGCPDiskTypesNoCredentialsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListGCPDiskTypesNoCredentialsOK creates a ListGCPDiskTypesNoCredentialsOK with default headers values
func NewListGCPDiskTypesNoCredentialsOK() *ListGCPDiskTypesNoCredentialsOK {
	return &ListGCPDiskTypesNoCredentialsOK{}
}

/*ListGCPDiskTypesNoCredentialsOK handles this case with default header values.

GCPDiskTypeList
*/
type ListGCPDiskTypesNoCredentialsOK struct {
	Payload models.GCPDiskTypeList
}

func (o *ListGCPDiskTypesNoCredentialsOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/gcp/disktypes][%d] listGCPDiskTypesNoCredentialsOK  %+v", 200, o.Payload)
}

func (o *ListGCPDiskTypesNoCredentialsOK) GetPayload() models.GCPDiskTypeList {
	return o.Payload
}

func (o *ListGCPDiskTypesNoCredentialsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListGCPDiskTypesNoCredentialsDefault creates a ListGCPDiskTypesNoCredentialsDefault with default headers values
func NewListGCPDiskTypesNoCredentialsDefault(code int) *ListGCPDiskTypesNoCredentialsDefault {
	return &ListGCPDiskTypesNoCredentialsDefault{
		_statusCode: code,
	}
}

/*ListGCPDiskTypesNoCredentialsDefault handles this case with default header values.

errorResponse
*/
type ListGCPDiskTypesNoCredentialsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list g c p disk types no credentials default response
func (o *ListGCPDiskTypesNoCredentialsDefault) Code() int {
	return o._statusCode
}

func (o *ListGCPDiskTypesNoCredentialsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/gcp/disktypes][%d] listGCPDiskTypesNoCredentials default  %+v", o._statusCode, o.Payload)
}

func (o *ListGCPDiskTypesNoCredentialsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListGCPDiskTypesNoCredentialsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
