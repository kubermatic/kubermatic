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

// ListAdminRuleGroupsReader is a Reader for the ListAdminRuleGroups structure.
type ListAdminRuleGroupsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListAdminRuleGroupsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListAdminRuleGroupsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListAdminRuleGroupsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListAdminRuleGroupsForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListAdminRuleGroupsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListAdminRuleGroupsOK creates a ListAdminRuleGroupsOK with default headers values
func NewListAdminRuleGroupsOK() *ListAdminRuleGroupsOK {
	return &ListAdminRuleGroupsOK{}
}

/* ListAdminRuleGroupsOK describes a response with status code 200, with default header values.

RuleGroup
*/
type ListAdminRuleGroupsOK struct {
	Payload []*models.RuleGroup
}

// IsSuccess returns true when this list admin rule groups o k response has a 2xx status code
func (o *ListAdminRuleGroupsOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list admin rule groups o k response has a 3xx status code
func (o *ListAdminRuleGroupsOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list admin rule groups o k response has a 4xx status code
func (o *ListAdminRuleGroupsOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list admin rule groups o k response has a 5xx status code
func (o *ListAdminRuleGroupsOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list admin rule groups o k response a status code equal to that given
func (o *ListAdminRuleGroupsOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListAdminRuleGroupsOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/seeds/{seed_name}/rulegroups][%d] listAdminRuleGroupsOK  %+v", 200, o.Payload)
}

func (o *ListAdminRuleGroupsOK) String() string {
	return fmt.Sprintf("[GET /api/v2/seeds/{seed_name}/rulegroups][%d] listAdminRuleGroupsOK  %+v", 200, o.Payload)
}

func (o *ListAdminRuleGroupsOK) GetPayload() []*models.RuleGroup {
	return o.Payload
}

func (o *ListAdminRuleGroupsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListAdminRuleGroupsUnauthorized creates a ListAdminRuleGroupsUnauthorized with default headers values
func NewListAdminRuleGroupsUnauthorized() *ListAdminRuleGroupsUnauthorized {
	return &ListAdminRuleGroupsUnauthorized{}
}

/* ListAdminRuleGroupsUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type ListAdminRuleGroupsUnauthorized struct {
}

// IsSuccess returns true when this list admin rule groups unauthorized response has a 2xx status code
func (o *ListAdminRuleGroupsUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list admin rule groups unauthorized response has a 3xx status code
func (o *ListAdminRuleGroupsUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list admin rule groups unauthorized response has a 4xx status code
func (o *ListAdminRuleGroupsUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this list admin rule groups unauthorized response has a 5xx status code
func (o *ListAdminRuleGroupsUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this list admin rule groups unauthorized response a status code equal to that given
func (o *ListAdminRuleGroupsUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *ListAdminRuleGroupsUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v2/seeds/{seed_name}/rulegroups][%d] listAdminRuleGroupsUnauthorized ", 401)
}

func (o *ListAdminRuleGroupsUnauthorized) String() string {
	return fmt.Sprintf("[GET /api/v2/seeds/{seed_name}/rulegroups][%d] listAdminRuleGroupsUnauthorized ", 401)
}

func (o *ListAdminRuleGroupsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListAdminRuleGroupsForbidden creates a ListAdminRuleGroupsForbidden with default headers values
func NewListAdminRuleGroupsForbidden() *ListAdminRuleGroupsForbidden {
	return &ListAdminRuleGroupsForbidden{}
}

/* ListAdminRuleGroupsForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type ListAdminRuleGroupsForbidden struct {
}

// IsSuccess returns true when this list admin rule groups forbidden response has a 2xx status code
func (o *ListAdminRuleGroupsForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list admin rule groups forbidden response has a 3xx status code
func (o *ListAdminRuleGroupsForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list admin rule groups forbidden response has a 4xx status code
func (o *ListAdminRuleGroupsForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this list admin rule groups forbidden response has a 5xx status code
func (o *ListAdminRuleGroupsForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this list admin rule groups forbidden response a status code equal to that given
func (o *ListAdminRuleGroupsForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *ListAdminRuleGroupsForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v2/seeds/{seed_name}/rulegroups][%d] listAdminRuleGroupsForbidden ", 403)
}

func (o *ListAdminRuleGroupsForbidden) String() string {
	return fmt.Sprintf("[GET /api/v2/seeds/{seed_name}/rulegroups][%d] listAdminRuleGroupsForbidden ", 403)
}

func (o *ListAdminRuleGroupsForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListAdminRuleGroupsDefault creates a ListAdminRuleGroupsDefault with default headers values
func NewListAdminRuleGroupsDefault(code int) *ListAdminRuleGroupsDefault {
	return &ListAdminRuleGroupsDefault{
		_statusCode: code,
	}
}

/* ListAdminRuleGroupsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListAdminRuleGroupsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list admin rule groups default response
func (o *ListAdminRuleGroupsDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list admin rule groups default response has a 2xx status code
func (o *ListAdminRuleGroupsDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list admin rule groups default response has a 3xx status code
func (o *ListAdminRuleGroupsDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list admin rule groups default response has a 4xx status code
func (o *ListAdminRuleGroupsDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list admin rule groups default response has a 5xx status code
func (o *ListAdminRuleGroupsDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list admin rule groups default response a status code equal to that given
func (o *ListAdminRuleGroupsDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListAdminRuleGroupsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/seeds/{seed_name}/rulegroups][%d] listAdminRuleGroups default  %+v", o._statusCode, o.Payload)
}

func (o *ListAdminRuleGroupsDefault) String() string {
	return fmt.Sprintf("[GET /api/v2/seeds/{seed_name}/rulegroups][%d] listAdminRuleGroups default  %+v", o._statusCode, o.Payload)
}

func (o *ListAdminRuleGroupsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListAdminRuleGroupsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
