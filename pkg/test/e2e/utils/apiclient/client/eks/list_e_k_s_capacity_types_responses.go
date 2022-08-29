// Code generated by go-swagger; DO NOT EDIT.

package eks

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListEKSCapacityTypesReader is a Reader for the ListEKSCapacityTypes structure.
type ListEKSCapacityTypesReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListEKSCapacityTypesReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListEKSCapacityTypesOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListEKSCapacityTypesUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListEKSCapacityTypesForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListEKSCapacityTypesDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListEKSCapacityTypesOK creates a ListEKSCapacityTypesOK with default headers values
func NewListEKSCapacityTypesOK() *ListEKSCapacityTypesOK {
	return &ListEKSCapacityTypesOK{}
}

/*
ListEKSCapacityTypesOK describes a response with status code 200, with default header values.

EKSCapacityTypeList
*/
type ListEKSCapacityTypesOK struct {
	Payload models.EKSCapacityTypeList
}

// IsSuccess returns true when this list e k s capacity types o k response has a 2xx status code
func (o *ListEKSCapacityTypesOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list e k s capacity types o k response has a 3xx status code
func (o *ListEKSCapacityTypesOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list e k s capacity types o k response has a 4xx status code
func (o *ListEKSCapacityTypesOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list e k s capacity types o k response has a 5xx status code
func (o *ListEKSCapacityTypesOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list e k s capacity types o k response a status code equal to that given
func (o *ListEKSCapacityTypesOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListEKSCapacityTypesOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/eks/capacitytypes][%d] listEKSCapacityTypesOK  %+v", 200, o.Payload)
}

func (o *ListEKSCapacityTypesOK) String() string {
	return fmt.Sprintf("[GET /api/v2/eks/capacitytypes][%d] listEKSCapacityTypesOK  %+v", 200, o.Payload)
}

func (o *ListEKSCapacityTypesOK) GetPayload() models.EKSCapacityTypeList {
	return o.Payload
}

func (o *ListEKSCapacityTypesOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListEKSCapacityTypesUnauthorized creates a ListEKSCapacityTypesUnauthorized with default headers values
func NewListEKSCapacityTypesUnauthorized() *ListEKSCapacityTypesUnauthorized {
	return &ListEKSCapacityTypesUnauthorized{}
}

/*
ListEKSCapacityTypesUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type ListEKSCapacityTypesUnauthorized struct {
}

// IsSuccess returns true when this list e k s capacity types unauthorized response has a 2xx status code
func (o *ListEKSCapacityTypesUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list e k s capacity types unauthorized response has a 3xx status code
func (o *ListEKSCapacityTypesUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list e k s capacity types unauthorized response has a 4xx status code
func (o *ListEKSCapacityTypesUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this list e k s capacity types unauthorized response has a 5xx status code
func (o *ListEKSCapacityTypesUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this list e k s capacity types unauthorized response a status code equal to that given
func (o *ListEKSCapacityTypesUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *ListEKSCapacityTypesUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v2/eks/capacitytypes][%d] listEKSCapacityTypesUnauthorized ", 401)
}

func (o *ListEKSCapacityTypesUnauthorized) String() string {
	return fmt.Sprintf("[GET /api/v2/eks/capacitytypes][%d] listEKSCapacityTypesUnauthorized ", 401)
}

func (o *ListEKSCapacityTypesUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListEKSCapacityTypesForbidden creates a ListEKSCapacityTypesForbidden with default headers values
func NewListEKSCapacityTypesForbidden() *ListEKSCapacityTypesForbidden {
	return &ListEKSCapacityTypesForbidden{}
}

/*
ListEKSCapacityTypesForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type ListEKSCapacityTypesForbidden struct {
}

// IsSuccess returns true when this list e k s capacity types forbidden response has a 2xx status code
func (o *ListEKSCapacityTypesForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list e k s capacity types forbidden response has a 3xx status code
func (o *ListEKSCapacityTypesForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list e k s capacity types forbidden response has a 4xx status code
func (o *ListEKSCapacityTypesForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this list e k s capacity types forbidden response has a 5xx status code
func (o *ListEKSCapacityTypesForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this list e k s capacity types forbidden response a status code equal to that given
func (o *ListEKSCapacityTypesForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *ListEKSCapacityTypesForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v2/eks/capacitytypes][%d] listEKSCapacityTypesForbidden ", 403)
}

func (o *ListEKSCapacityTypesForbidden) String() string {
	return fmt.Sprintf("[GET /api/v2/eks/capacitytypes][%d] listEKSCapacityTypesForbidden ", 403)
}

func (o *ListEKSCapacityTypesForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListEKSCapacityTypesDefault creates a ListEKSCapacityTypesDefault with default headers values
func NewListEKSCapacityTypesDefault(code int) *ListEKSCapacityTypesDefault {
	return &ListEKSCapacityTypesDefault{
		_statusCode: code,
	}
}

/*
ListEKSCapacityTypesDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListEKSCapacityTypesDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list e k s capacity types default response
func (o *ListEKSCapacityTypesDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list e k s capacity types default response has a 2xx status code
func (o *ListEKSCapacityTypesDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list e k s capacity types default response has a 3xx status code
func (o *ListEKSCapacityTypesDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list e k s capacity types default response has a 4xx status code
func (o *ListEKSCapacityTypesDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list e k s capacity types default response has a 5xx status code
func (o *ListEKSCapacityTypesDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list e k s capacity types default response a status code equal to that given
func (o *ListEKSCapacityTypesDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListEKSCapacityTypesDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/eks/capacitytypes][%d] listEKSCapacityTypes default  %+v", o._statusCode, o.Payload)
}

func (o *ListEKSCapacityTypesDefault) String() string {
	return fmt.Sprintf("[GET /api/v2/eks/capacitytypes][%d] listEKSCapacityTypes default  %+v", o._statusCode, o.Payload)
}

func (o *ListEKSCapacityTypesDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListEKSCapacityTypesDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
