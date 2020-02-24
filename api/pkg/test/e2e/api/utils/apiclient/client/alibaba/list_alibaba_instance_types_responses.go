// Code generated by go-swagger; DO NOT EDIT.

package alibaba

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// ListAlibabaInstanceTypesReader is a Reader for the ListAlibabaInstanceTypes structure.
type ListAlibabaInstanceTypesReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListAlibabaInstanceTypesReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListAlibabaInstanceTypesOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListAlibabaInstanceTypesDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListAlibabaInstanceTypesOK creates a ListAlibabaInstanceTypesOK with default headers values
func NewListAlibabaInstanceTypesOK() *ListAlibabaInstanceTypesOK {
	return &ListAlibabaInstanceTypesOK{}
}

/*ListAlibabaInstanceTypesOK handles this case with default header values.

AlibabaInstanceTypeList
*/
type ListAlibabaInstanceTypesOK struct {
	Payload models.AlibabaInstanceTypeList
}

func (o *ListAlibabaInstanceTypesOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/providers/alibaba/instancetypes][%d] listAlibabaInstanceTypesOK  %+v", 200, o.Payload)
}

func (o *ListAlibabaInstanceTypesOK) GetPayload() models.AlibabaInstanceTypeList {
	return o.Payload
}

func (o *ListAlibabaInstanceTypesOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListAlibabaInstanceTypesDefault creates a ListAlibabaInstanceTypesDefault with default headers values
func NewListAlibabaInstanceTypesDefault(code int) *ListAlibabaInstanceTypesDefault {
	return &ListAlibabaInstanceTypesDefault{
		_statusCode: code,
	}
}

/*ListAlibabaInstanceTypesDefault handles this case with default header values.

errorResponse
*/
type ListAlibabaInstanceTypesDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list alibaba instance types default response
func (o *ListAlibabaInstanceTypesDefault) Code() int {
	return o._statusCode
}

func (o *ListAlibabaInstanceTypesDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/providers/alibaba/instancetypes][%d] listAlibabaInstanceTypes default  %+v", o._statusCode, o.Payload)
}

func (o *ListAlibabaInstanceTypesDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListAlibabaInstanceTypesDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
