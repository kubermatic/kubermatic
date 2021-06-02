// Code generated by go-swagger; DO NOT EDIT.

package rulegroup

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// CreateRuleGroupReader is a Reader for the CreateRuleGroup structure.
type CreateRuleGroupReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *CreateRuleGroupReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 201:
		result := NewCreateRuleGroupCreated()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewCreateRuleGroupUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewCreateRuleGroupForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewCreateRuleGroupDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewCreateRuleGroupCreated creates a CreateRuleGroupCreated with default headers values
func NewCreateRuleGroupCreated() *CreateRuleGroupCreated {
	return &CreateRuleGroupCreated{}
}

/*CreateRuleGroupCreated handles this case with default header values.

RuleGroup
*/
type CreateRuleGroupCreated struct {
	Payload *models.RuleGroup
}

func (o *CreateRuleGroupCreated) Error() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/rule_groups][%d] createRuleGroupCreated  %+v", 201, o.Payload)
}

func (o *CreateRuleGroupCreated) GetPayload() *models.RuleGroup {
	return o.Payload
}

func (o *CreateRuleGroupCreated) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.RuleGroup)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewCreateRuleGroupUnauthorized creates a CreateRuleGroupUnauthorized with default headers values
func NewCreateRuleGroupUnauthorized() *CreateRuleGroupUnauthorized {
	return &CreateRuleGroupUnauthorized{}
}

/*CreateRuleGroupUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type CreateRuleGroupUnauthorized struct {
}

func (o *CreateRuleGroupUnauthorized) Error() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/rule_groups][%d] createRuleGroupUnauthorized ", 401)
}

func (o *CreateRuleGroupUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateRuleGroupForbidden creates a CreateRuleGroupForbidden with default headers values
func NewCreateRuleGroupForbidden() *CreateRuleGroupForbidden {
	return &CreateRuleGroupForbidden{}
}

/*CreateRuleGroupForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type CreateRuleGroupForbidden struct {
}

func (o *CreateRuleGroupForbidden) Error() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/rule_groups][%d] createRuleGroupForbidden ", 403)
}

func (o *CreateRuleGroupForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateRuleGroupDefault creates a CreateRuleGroupDefault with default headers values
func NewCreateRuleGroupDefault(code int) *CreateRuleGroupDefault {
	return &CreateRuleGroupDefault{
		_statusCode: code,
	}
}

/*CreateRuleGroupDefault handles this case with default header values.

errorResponse
*/
type CreateRuleGroupDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the create rule group default response
func (o *CreateRuleGroupDefault) Code() int {
	return o._statusCode
}

func (o *CreateRuleGroupDefault) Error() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/rule_groups][%d] createRuleGroup default  %+v", o._statusCode, o.Payload)
}

func (o *CreateRuleGroupDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *CreateRuleGroupDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
