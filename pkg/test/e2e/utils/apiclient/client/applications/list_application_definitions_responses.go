// Code generated by go-swagger; DO NOT EDIT.

package applications

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListApplicationDefinitionsReader is a Reader for the ListApplicationDefinitions structure.
type ListApplicationDefinitionsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListApplicationDefinitionsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListApplicationDefinitionsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListApplicationDefinitionsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListApplicationDefinitionsForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListApplicationDefinitionsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListApplicationDefinitionsOK creates a ListApplicationDefinitionsOK with default headers values
func NewListApplicationDefinitionsOK() *ListApplicationDefinitionsOK {
	return &ListApplicationDefinitionsOK{}
}

/* ListApplicationDefinitionsOK describes a response with status code 200, with default header values.

ApplicationDefinition
*/
type ListApplicationDefinitionsOK struct {
	Payload []*models.ApplicationDefinition
}

// IsSuccess returns true when this list application definitions o k response has a 2xx status code
func (o *ListApplicationDefinitionsOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list application definitions o k response has a 3xx status code
func (o *ListApplicationDefinitionsOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list application definitions o k response has a 4xx status code
func (o *ListApplicationDefinitionsOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list application definitions o k response has a 5xx status code
func (o *ListApplicationDefinitionsOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list application definitions o k response a status code equal to that given
func (o *ListApplicationDefinitionsOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListApplicationDefinitionsOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/applicationdefinitions][%d] listApplicationDefinitionsOK  %+v", 200, o.Payload)
}

func (o *ListApplicationDefinitionsOK) String() string {
	return fmt.Sprintf("[GET /api/v2/applicationdefinitions][%d] listApplicationDefinitionsOK  %+v", 200, o.Payload)
}

func (o *ListApplicationDefinitionsOK) GetPayload() []*models.ApplicationDefinition {
	return o.Payload
}

func (o *ListApplicationDefinitionsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListApplicationDefinitionsUnauthorized creates a ListApplicationDefinitionsUnauthorized with default headers values
func NewListApplicationDefinitionsUnauthorized() *ListApplicationDefinitionsUnauthorized {
	return &ListApplicationDefinitionsUnauthorized{}
}

/* ListApplicationDefinitionsUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type ListApplicationDefinitionsUnauthorized struct {
}

// IsSuccess returns true when this list application definitions unauthorized response has a 2xx status code
func (o *ListApplicationDefinitionsUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list application definitions unauthorized response has a 3xx status code
func (o *ListApplicationDefinitionsUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list application definitions unauthorized response has a 4xx status code
func (o *ListApplicationDefinitionsUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this list application definitions unauthorized response has a 5xx status code
func (o *ListApplicationDefinitionsUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this list application definitions unauthorized response a status code equal to that given
func (o *ListApplicationDefinitionsUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *ListApplicationDefinitionsUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v2/applicationdefinitions][%d] listApplicationDefinitionsUnauthorized ", 401)
}

func (o *ListApplicationDefinitionsUnauthorized) String() string {
	return fmt.Sprintf("[GET /api/v2/applicationdefinitions][%d] listApplicationDefinitionsUnauthorized ", 401)
}

func (o *ListApplicationDefinitionsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListApplicationDefinitionsForbidden creates a ListApplicationDefinitionsForbidden with default headers values
func NewListApplicationDefinitionsForbidden() *ListApplicationDefinitionsForbidden {
	return &ListApplicationDefinitionsForbidden{}
}

/* ListApplicationDefinitionsForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type ListApplicationDefinitionsForbidden struct {
}

// IsSuccess returns true when this list application definitions forbidden response has a 2xx status code
func (o *ListApplicationDefinitionsForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list application definitions forbidden response has a 3xx status code
func (o *ListApplicationDefinitionsForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list application definitions forbidden response has a 4xx status code
func (o *ListApplicationDefinitionsForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this list application definitions forbidden response has a 5xx status code
func (o *ListApplicationDefinitionsForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this list application definitions forbidden response a status code equal to that given
func (o *ListApplicationDefinitionsForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *ListApplicationDefinitionsForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v2/applicationdefinitions][%d] listApplicationDefinitionsForbidden ", 403)
}

func (o *ListApplicationDefinitionsForbidden) String() string {
	return fmt.Sprintf("[GET /api/v2/applicationdefinitions][%d] listApplicationDefinitionsForbidden ", 403)
}

func (o *ListApplicationDefinitionsForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListApplicationDefinitionsDefault creates a ListApplicationDefinitionsDefault with default headers values
func NewListApplicationDefinitionsDefault(code int) *ListApplicationDefinitionsDefault {
	return &ListApplicationDefinitionsDefault{
		_statusCode: code,
	}
}

/* ListApplicationDefinitionsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListApplicationDefinitionsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list application definitions default response
func (o *ListApplicationDefinitionsDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list application definitions default response has a 2xx status code
func (o *ListApplicationDefinitionsDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list application definitions default response has a 3xx status code
func (o *ListApplicationDefinitionsDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list application definitions default response has a 4xx status code
func (o *ListApplicationDefinitionsDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list application definitions default response has a 5xx status code
func (o *ListApplicationDefinitionsDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list application definitions default response a status code equal to that given
func (o *ListApplicationDefinitionsDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListApplicationDefinitionsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/applicationdefinitions][%d] listApplicationDefinitions default  %+v", o._statusCode, o.Payload)
}

func (o *ListApplicationDefinitionsDefault) String() string {
	return fmt.Sprintf("[GET /api/v2/applicationdefinitions][%d] listApplicationDefinitions default  %+v", o._statusCode, o.Payload)
}

func (o *ListApplicationDefinitionsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListApplicationDefinitionsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
