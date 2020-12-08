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

// ListOpenstackSecurityGroupsNoCredentialsV2Reader is a Reader for the ListOpenstackSecurityGroupsNoCredentialsV2 structure.
type ListOpenstackSecurityGroupsNoCredentialsV2Reader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListOpenstackSecurityGroupsNoCredentialsV2Reader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListOpenstackSecurityGroupsNoCredentialsV2OK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListOpenstackSecurityGroupsNoCredentialsV2Default(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListOpenstackSecurityGroupsNoCredentialsV2OK creates a ListOpenstackSecurityGroupsNoCredentialsV2OK with default headers values
func NewListOpenstackSecurityGroupsNoCredentialsV2OK() *ListOpenstackSecurityGroupsNoCredentialsV2OK {
	return &ListOpenstackSecurityGroupsNoCredentialsV2OK{}
}

/*ListOpenstackSecurityGroupsNoCredentialsV2OK handles this case with default header values.

OpenstackSecurityGroup
*/
type ListOpenstackSecurityGroupsNoCredentialsV2OK struct {
	Payload []*models.OpenstackSecurityGroup
}

func (o *ListOpenstackSecurityGroupsNoCredentialsV2OK) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/openstack/securitygroups][%d] listOpenstackSecurityGroupsNoCredentialsV2OK  %+v", 200, o.Payload)
}

func (o *ListOpenstackSecurityGroupsNoCredentialsV2OK) GetPayload() []*models.OpenstackSecurityGroup {
	return o.Payload
}

func (o *ListOpenstackSecurityGroupsNoCredentialsV2OK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListOpenstackSecurityGroupsNoCredentialsV2Default creates a ListOpenstackSecurityGroupsNoCredentialsV2Default with default headers values
func NewListOpenstackSecurityGroupsNoCredentialsV2Default(code int) *ListOpenstackSecurityGroupsNoCredentialsV2Default {
	return &ListOpenstackSecurityGroupsNoCredentialsV2Default{
		_statusCode: code,
	}
}

/*ListOpenstackSecurityGroupsNoCredentialsV2Default handles this case with default header values.

errorResponse
*/
type ListOpenstackSecurityGroupsNoCredentialsV2Default struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list openstack security groups no credentials v2 default response
func (o *ListOpenstackSecurityGroupsNoCredentialsV2Default) Code() int {
	return o._statusCode
}

func (o *ListOpenstackSecurityGroupsNoCredentialsV2Default) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/openstack/securitygroups][%d] listOpenstackSecurityGroupsNoCredentialsV2 default  %+v", o._statusCode, o.Payload)
}

func (o *ListOpenstackSecurityGroupsNoCredentialsV2Default) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListOpenstackSecurityGroupsNoCredentialsV2Default) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
