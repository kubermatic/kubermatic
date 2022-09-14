// Code generated by go-swagger; DO NOT EDIT.

package aks

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListAKSResourceGroupsReader is a Reader for the ListAKSResourceGroups structure.
type ListAKSResourceGroupsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListAKSResourceGroupsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListAKSResourceGroupsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListAKSResourceGroupsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListAKSResourceGroupsOK creates a ListAKSResourceGroupsOK with default headers values
func NewListAKSResourceGroupsOK() *ListAKSResourceGroupsOK {
	return &ListAKSResourceGroupsOK{}
}

/*
ListAKSResourceGroupsOK describes a response with status code 200, with default header values.

AzureResourceGroupList
*/
type ListAKSResourceGroupsOK struct {
	Payload models.AzureResourceGroupList
}

// IsSuccess returns true when this list a k s resource groups o k response has a 2xx status code
func (o *ListAKSResourceGroupsOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list a k s resource groups o k response has a 3xx status code
func (o *ListAKSResourceGroupsOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list a k s resource groups o k response has a 4xx status code
func (o *ListAKSResourceGroupsOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list a k s resource groups o k response has a 5xx status code
func (o *ListAKSResourceGroupsOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list a k s resource groups o k response a status code equal to that given
func (o *ListAKSResourceGroupsOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListAKSResourceGroupsOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/aks/resourcegroups][%d] listAKSResourceGroupsOK  %+v", 200, o.Payload)
}

func (o *ListAKSResourceGroupsOK) String() string {
	return fmt.Sprintf("[GET /api/v2/providers/aks/resourcegroups][%d] listAKSResourceGroupsOK  %+v", 200, o.Payload)
}

func (o *ListAKSResourceGroupsOK) GetPayload() models.AzureResourceGroupList {
	return o.Payload
}

func (o *ListAKSResourceGroupsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListAKSResourceGroupsDefault creates a ListAKSResourceGroupsDefault with default headers values
func NewListAKSResourceGroupsDefault(code int) *ListAKSResourceGroupsDefault {
	return &ListAKSResourceGroupsDefault{
		_statusCode: code,
	}
}

/*
ListAKSResourceGroupsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListAKSResourceGroupsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list a k s resource groups default response
func (o *ListAKSResourceGroupsDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list a k s resource groups default response has a 2xx status code
func (o *ListAKSResourceGroupsDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list a k s resource groups default response has a 3xx status code
func (o *ListAKSResourceGroupsDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list a k s resource groups default response has a 4xx status code
func (o *ListAKSResourceGroupsDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list a k s resource groups default response has a 5xx status code
func (o *ListAKSResourceGroupsDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list a k s resource groups default response a status code equal to that given
func (o *ListAKSResourceGroupsDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListAKSResourceGroupsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/aks/resourcegroups][%d] listAKSResourceGroups default  %+v", o._statusCode, o.Payload)
}

func (o *ListAKSResourceGroupsDefault) String() string {
	return fmt.Sprintf("[GET /api/v2/providers/aks/resourcegroups][%d] listAKSResourceGroups default  %+v", o._statusCode, o.Payload)
}

func (o *ListAKSResourceGroupsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListAKSResourceGroupsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
