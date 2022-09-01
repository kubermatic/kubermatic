// Code generated by go-swagger; DO NOT EDIT.

package vmwareclouddirector

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListVMwareCloudDirectorStorageProfilesNoCredentialsReader is a Reader for the ListVMwareCloudDirectorStorageProfilesNoCredentials structure.
type ListVMwareCloudDirectorStorageProfilesNoCredentialsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListVMwareCloudDirectorStorageProfilesNoCredentialsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListVMwareCloudDirectorStorageProfilesNoCredentialsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListVMwareCloudDirectorStorageProfilesNoCredentialsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListVMwareCloudDirectorStorageProfilesNoCredentialsOK creates a ListVMwareCloudDirectorStorageProfilesNoCredentialsOK with default headers values
func NewListVMwareCloudDirectorStorageProfilesNoCredentialsOK() *ListVMwareCloudDirectorStorageProfilesNoCredentialsOK {
	return &ListVMwareCloudDirectorStorageProfilesNoCredentialsOK{}
}

/* ListVMwareCloudDirectorStorageProfilesNoCredentialsOK describes a response with status code 200, with default header values.

VMwareCloudDirectorStorageProfileList
*/
type ListVMwareCloudDirectorStorageProfilesNoCredentialsOK struct {
	Payload models.VMwareCloudDirectorStorageProfileList
}

// IsSuccess returns true when this list v mware cloud director storage profiles no credentials o k response has a 2xx status code
func (o *ListVMwareCloudDirectorStorageProfilesNoCredentialsOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list v mware cloud director storage profiles no credentials o k response has a 3xx status code
func (o *ListVMwareCloudDirectorStorageProfilesNoCredentialsOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list v mware cloud director storage profiles no credentials o k response has a 4xx status code
func (o *ListVMwareCloudDirectorStorageProfilesNoCredentialsOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list v mware cloud director storage profiles no credentials o k response has a 5xx status code
func (o *ListVMwareCloudDirectorStorageProfilesNoCredentialsOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list v mware cloud director storage profiles no credentials o k response a status code equal to that given
func (o *ListVMwareCloudDirectorStorageProfilesNoCredentialsOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListVMwareCloudDirectorStorageProfilesNoCredentialsOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/vmwareclouddirector/storageprofiles][%d] listVMwareCloudDirectorStorageProfilesNoCredentialsOK  %+v", 200, o.Payload)
}

func (o *ListVMwareCloudDirectorStorageProfilesNoCredentialsOK) String() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/vmwareclouddirector/storageprofiles][%d] listVMwareCloudDirectorStorageProfilesNoCredentialsOK  %+v", 200, o.Payload)
}

func (o *ListVMwareCloudDirectorStorageProfilesNoCredentialsOK) GetPayload() models.VMwareCloudDirectorStorageProfileList {
	return o.Payload
}

func (o *ListVMwareCloudDirectorStorageProfilesNoCredentialsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListVMwareCloudDirectorStorageProfilesNoCredentialsDefault creates a ListVMwareCloudDirectorStorageProfilesNoCredentialsDefault with default headers values
func NewListVMwareCloudDirectorStorageProfilesNoCredentialsDefault(code int) *ListVMwareCloudDirectorStorageProfilesNoCredentialsDefault {
	return &ListVMwareCloudDirectorStorageProfilesNoCredentialsDefault{
		_statusCode: code,
	}
}

/* ListVMwareCloudDirectorStorageProfilesNoCredentialsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListVMwareCloudDirectorStorageProfilesNoCredentialsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list v mware cloud director storage profiles no credentials default response
func (o *ListVMwareCloudDirectorStorageProfilesNoCredentialsDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list v mware cloud director storage profiles no credentials default response has a 2xx status code
func (o *ListVMwareCloudDirectorStorageProfilesNoCredentialsDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list v mware cloud director storage profiles no credentials default response has a 3xx status code
func (o *ListVMwareCloudDirectorStorageProfilesNoCredentialsDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list v mware cloud director storage profiles no credentials default response has a 4xx status code
func (o *ListVMwareCloudDirectorStorageProfilesNoCredentialsDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list v mware cloud director storage profiles no credentials default response has a 5xx status code
func (o *ListVMwareCloudDirectorStorageProfilesNoCredentialsDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list v mware cloud director storage profiles no credentials default response a status code equal to that given
func (o *ListVMwareCloudDirectorStorageProfilesNoCredentialsDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListVMwareCloudDirectorStorageProfilesNoCredentialsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/vmwareclouddirector/storageprofiles][%d] listVMwareCloudDirectorStorageProfilesNoCredentials default  %+v", o._statusCode, o.Payload)
}

func (o *ListVMwareCloudDirectorStorageProfilesNoCredentialsDefault) String() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/vmwareclouddirector/storageprofiles][%d] listVMwareCloudDirectorStorageProfilesNoCredentials default  %+v", o._statusCode, o.Payload)
}

func (o *ListVMwareCloudDirectorStorageProfilesNoCredentialsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListVMwareCloudDirectorStorageProfilesNoCredentialsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
