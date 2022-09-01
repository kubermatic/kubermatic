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

/* CreateRuleGroupCreated describes a response with status code 201, with default header values.

RuleGroup
*/
type CreateRuleGroupCreated struct {
	Payload *models.RuleGroup
}

// IsSuccess returns true when this create rule group created response has a 2xx status code
func (o *CreateRuleGroupCreated) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this create rule group created response has a 3xx status code
func (o *CreateRuleGroupCreated) IsRedirect() bool {
	return false
}

// IsClientError returns true when this create rule group created response has a 4xx status code
func (o *CreateRuleGroupCreated) IsClientError() bool {
	return false
}

// IsServerError returns true when this create rule group created response has a 5xx status code
func (o *CreateRuleGroupCreated) IsServerError() bool {
	return false
}

// IsCode returns true when this create rule group created response a status code equal to that given
func (o *CreateRuleGroupCreated) IsCode(code int) bool {
	return code == 201
}

func (o *CreateRuleGroupCreated) Error() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/rulegroups][%d] createRuleGroupCreated  %+v", 201, o.Payload)
}

func (o *CreateRuleGroupCreated) String() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/rulegroups][%d] createRuleGroupCreated  %+v", 201, o.Payload)
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

/* CreateRuleGroupUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type CreateRuleGroupUnauthorized struct {
}

// IsSuccess returns true when this create rule group unauthorized response has a 2xx status code
func (o *CreateRuleGroupUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this create rule group unauthorized response has a 3xx status code
func (o *CreateRuleGroupUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this create rule group unauthorized response has a 4xx status code
func (o *CreateRuleGroupUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this create rule group unauthorized response has a 5xx status code
func (o *CreateRuleGroupUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this create rule group unauthorized response a status code equal to that given
func (o *CreateRuleGroupUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *CreateRuleGroupUnauthorized) Error() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/rulegroups][%d] createRuleGroupUnauthorized ", 401)
}

func (o *CreateRuleGroupUnauthorized) String() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/rulegroups][%d] createRuleGroupUnauthorized ", 401)
}

func (o *CreateRuleGroupUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateRuleGroupForbidden creates a CreateRuleGroupForbidden with default headers values
func NewCreateRuleGroupForbidden() *CreateRuleGroupForbidden {
	return &CreateRuleGroupForbidden{}
}

/* CreateRuleGroupForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type CreateRuleGroupForbidden struct {
}

// IsSuccess returns true when this create rule group forbidden response has a 2xx status code
func (o *CreateRuleGroupForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this create rule group forbidden response has a 3xx status code
func (o *CreateRuleGroupForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this create rule group forbidden response has a 4xx status code
func (o *CreateRuleGroupForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this create rule group forbidden response has a 5xx status code
func (o *CreateRuleGroupForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this create rule group forbidden response a status code equal to that given
func (o *CreateRuleGroupForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *CreateRuleGroupForbidden) Error() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/rulegroups][%d] createRuleGroupForbidden ", 403)
}

func (o *CreateRuleGroupForbidden) String() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/rulegroups][%d] createRuleGroupForbidden ", 403)
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

/* CreateRuleGroupDefault describes a response with status code -1, with default header values.

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

// IsSuccess returns true when this create rule group default response has a 2xx status code
func (o *CreateRuleGroupDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this create rule group default response has a 3xx status code
func (o *CreateRuleGroupDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this create rule group default response has a 4xx status code
func (o *CreateRuleGroupDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this create rule group default response has a 5xx status code
func (o *CreateRuleGroupDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this create rule group default response a status code equal to that given
func (o *CreateRuleGroupDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *CreateRuleGroupDefault) Error() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/rulegroups][%d] createRuleGroup default  %+v", o._statusCode, o.Payload)
}

func (o *CreateRuleGroupDefault) String() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/rulegroups][%d] createRuleGroup default  %+v", o._statusCode, o.Payload)
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
