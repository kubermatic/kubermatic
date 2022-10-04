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

// GetApplicationDefinitionReader is a Reader for the GetApplicationDefinition structure.
type GetApplicationDefinitionReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetApplicationDefinitionReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetApplicationDefinitionOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewGetApplicationDefinitionUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewGetApplicationDefinitionForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewGetApplicationDefinitionDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetApplicationDefinitionOK creates a GetApplicationDefinitionOK with default headers values
func NewGetApplicationDefinitionOK() *GetApplicationDefinitionOK {
	return &GetApplicationDefinitionOK{}
}

/*
GetApplicationDefinitionOK describes a response with status code 200, with default header values.

ApplicationDefinition
*/
type GetApplicationDefinitionOK struct {
	Payload *models.ApplicationDefinition
}

// IsSuccess returns true when this get application definition o k response has a 2xx status code
func (o *GetApplicationDefinitionOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this get application definition o k response has a 3xx status code
func (o *GetApplicationDefinitionOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get application definition o k response has a 4xx status code
func (o *GetApplicationDefinitionOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this get application definition o k response has a 5xx status code
func (o *GetApplicationDefinitionOK) IsServerError() bool {
	return false
}

// IsCode returns true when this get application definition o k response a status code equal to that given
func (o *GetApplicationDefinitionOK) IsCode(code int) bool {
	return code == 200
}

func (o *GetApplicationDefinitionOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/applicationdefinitions/{appdef_name}][%d] getApplicationDefinitionOK  %+v", 200, o.Payload)
}

func (o *GetApplicationDefinitionOK) String() string {
	return fmt.Sprintf("[GET /api/v2/applicationdefinitions/{appdef_name}][%d] getApplicationDefinitionOK  %+v", 200, o.Payload)
}

func (o *GetApplicationDefinitionOK) GetPayload() *models.ApplicationDefinition {
	return o.Payload
}

func (o *GetApplicationDefinitionOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ApplicationDefinition)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetApplicationDefinitionUnauthorized creates a GetApplicationDefinitionUnauthorized with default headers values
func NewGetApplicationDefinitionUnauthorized() *GetApplicationDefinitionUnauthorized {
	return &GetApplicationDefinitionUnauthorized{}
}

/*
GetApplicationDefinitionUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type GetApplicationDefinitionUnauthorized struct {
}

// IsSuccess returns true when this get application definition unauthorized response has a 2xx status code
func (o *GetApplicationDefinitionUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this get application definition unauthorized response has a 3xx status code
func (o *GetApplicationDefinitionUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get application definition unauthorized response has a 4xx status code
func (o *GetApplicationDefinitionUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this get application definition unauthorized response has a 5xx status code
func (o *GetApplicationDefinitionUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this get application definition unauthorized response a status code equal to that given
func (o *GetApplicationDefinitionUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *GetApplicationDefinitionUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v2/applicationdefinitions/{appdef_name}][%d] getApplicationDefinitionUnauthorized ", 401)
}

func (o *GetApplicationDefinitionUnauthorized) String() string {
	return fmt.Sprintf("[GET /api/v2/applicationdefinitions/{appdef_name}][%d] getApplicationDefinitionUnauthorized ", 401)
}

func (o *GetApplicationDefinitionUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetApplicationDefinitionForbidden creates a GetApplicationDefinitionForbidden with default headers values
func NewGetApplicationDefinitionForbidden() *GetApplicationDefinitionForbidden {
	return &GetApplicationDefinitionForbidden{}
}

/*
GetApplicationDefinitionForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type GetApplicationDefinitionForbidden struct {
}

// IsSuccess returns true when this get application definition forbidden response has a 2xx status code
func (o *GetApplicationDefinitionForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this get application definition forbidden response has a 3xx status code
func (o *GetApplicationDefinitionForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get application definition forbidden response has a 4xx status code
func (o *GetApplicationDefinitionForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this get application definition forbidden response has a 5xx status code
func (o *GetApplicationDefinitionForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this get application definition forbidden response a status code equal to that given
func (o *GetApplicationDefinitionForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *GetApplicationDefinitionForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v2/applicationdefinitions/{appdef_name}][%d] getApplicationDefinitionForbidden ", 403)
}

func (o *GetApplicationDefinitionForbidden) String() string {
	return fmt.Sprintf("[GET /api/v2/applicationdefinitions/{appdef_name}][%d] getApplicationDefinitionForbidden ", 403)
}

func (o *GetApplicationDefinitionForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetApplicationDefinitionDefault creates a GetApplicationDefinitionDefault with default headers values
func NewGetApplicationDefinitionDefault(code int) *GetApplicationDefinitionDefault {
	return &GetApplicationDefinitionDefault{
		_statusCode: code,
	}
}

/*
GetApplicationDefinitionDefault describes a response with status code -1, with default header values.

errorResponse
*/
type GetApplicationDefinitionDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the get application definition default response
func (o *GetApplicationDefinitionDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this get application definition default response has a 2xx status code
func (o *GetApplicationDefinitionDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this get application definition default response has a 3xx status code
func (o *GetApplicationDefinitionDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this get application definition default response has a 4xx status code
func (o *GetApplicationDefinitionDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this get application definition default response has a 5xx status code
func (o *GetApplicationDefinitionDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this get application definition default response a status code equal to that given
func (o *GetApplicationDefinitionDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *GetApplicationDefinitionDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/applicationdefinitions/{appdef_name}][%d] getApplicationDefinition default  %+v", o._statusCode, o.Payload)
}

func (o *GetApplicationDefinitionDefault) String() string {
	return fmt.Sprintf("[GET /api/v2/applicationdefinitions/{appdef_name}][%d] getApplicationDefinition default  %+v", o._statusCode, o.Payload)
}

func (o *GetApplicationDefinitionDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *GetApplicationDefinitionDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
