// Code generated by go-swagger; DO NOT EDIT.

package openstack

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/api/utils/apiclient/models"
)

// ListOpenstackSizesNoCredentialsReader is a Reader for the ListOpenstackSizesNoCredentials structure.
type ListOpenstackSizesNoCredentialsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListOpenstackSizesNoCredentialsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListOpenstackSizesNoCredentialsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListOpenstackSizesNoCredentialsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListOpenstackSizesNoCredentialsOK creates a ListOpenstackSizesNoCredentialsOK with default headers values
func NewListOpenstackSizesNoCredentialsOK() *ListOpenstackSizesNoCredentialsOK {
	return &ListOpenstackSizesNoCredentialsOK{}
}

/*ListOpenstackSizesNoCredentialsOK handles this case with default header values.

OpenstackSize
*/
type ListOpenstackSizesNoCredentialsOK struct {
	Payload []*models.OpenstackSize
}

func (o *ListOpenstackSizesNoCredentialsOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/openstack/sizes][%d] listOpenstackSizesNoCredentialsOK  %+v", 200, o.Payload)
}

func (o *ListOpenstackSizesNoCredentialsOK) GetPayload() []*models.OpenstackSize {
	return o.Payload
}

func (o *ListOpenstackSizesNoCredentialsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListOpenstackSizesNoCredentialsDefault creates a ListOpenstackSizesNoCredentialsDefault with default headers values
func NewListOpenstackSizesNoCredentialsDefault(code int) *ListOpenstackSizesNoCredentialsDefault {
	return &ListOpenstackSizesNoCredentialsDefault{
		_statusCode: code,
	}
}

/*ListOpenstackSizesNoCredentialsDefault handles this case with default header values.

errorResponse
*/
type ListOpenstackSizesNoCredentialsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list openstack sizes no credentials default response
func (o *ListOpenstackSizesNoCredentialsDefault) Code() int {
	return o._statusCode
}

func (o *ListOpenstackSizesNoCredentialsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/openstack/sizes][%d] listOpenstackSizesNoCredentials default  %+v", o._statusCode, o.Payload)
}

func (o *ListOpenstackSizesNoCredentialsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListOpenstackSizesNoCredentialsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
