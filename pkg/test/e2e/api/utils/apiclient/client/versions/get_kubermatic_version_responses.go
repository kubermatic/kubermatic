// Code generated by go-swagger; DO NOT EDIT.

package versions

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/models"
)

// GetKubermaticVersionReader is a Reader for the GetKubermaticVersion structure.
type GetKubermaticVersionReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetKubermaticVersionReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetKubermaticVersionOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewGetKubermaticVersionDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetKubermaticVersionOK creates a GetKubermaticVersionOK with default headers values
func NewGetKubermaticVersionOK() *GetKubermaticVersionOK {
	return &GetKubermaticVersionOK{}
}

/*GetKubermaticVersionOK handles this case with default header values.

KubermaticVersions
*/
type GetKubermaticVersionOK struct {
	Payload *models.KubermaticVersions
}

func (o *GetKubermaticVersionOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/version][%d] getKubermaticVersionOK  %+v", 200, o.Payload)
}

func (o *GetKubermaticVersionOK) GetPayload() *models.KubermaticVersions {
	return o.Payload
}

func (o *GetKubermaticVersionOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.KubermaticVersions)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetKubermaticVersionDefault creates a GetKubermaticVersionDefault with default headers values
func NewGetKubermaticVersionDefault(code int) *GetKubermaticVersionDefault {
	return &GetKubermaticVersionDefault{
		_statusCode: code,
	}
}

/*GetKubermaticVersionDefault handles this case with default header values.

errorResponse
*/
type GetKubermaticVersionDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the get kubermatic version default response
func (o *GetKubermaticVersionDefault) Code() int {
	return o._statusCode
}

func (o *GetKubermaticVersionDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/version][%d] getKubermaticVersion default  %+v", o._statusCode, o.Payload)
}

func (o *GetKubermaticVersionDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *GetKubermaticVersionDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
