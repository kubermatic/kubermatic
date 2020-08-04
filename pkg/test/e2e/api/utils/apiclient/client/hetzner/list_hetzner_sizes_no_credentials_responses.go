// Code generated by go-swagger; DO NOT EDIT.

package hetzner

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/api/utils/apiclient/models"
)

// ListHetznerSizesNoCredentialsReader is a Reader for the ListHetznerSizesNoCredentials structure.
type ListHetznerSizesNoCredentialsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListHetznerSizesNoCredentialsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListHetznerSizesNoCredentialsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListHetznerSizesNoCredentialsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListHetznerSizesNoCredentialsOK creates a ListHetznerSizesNoCredentialsOK with default headers values
func NewListHetznerSizesNoCredentialsOK() *ListHetznerSizesNoCredentialsOK {
	return &ListHetznerSizesNoCredentialsOK{}
}

/*ListHetznerSizesNoCredentialsOK handles this case with default header values.

HetznerSizeList
*/
type ListHetznerSizesNoCredentialsOK struct {
	Payload *models.HetznerSizeList
}

func (o *ListHetznerSizesNoCredentialsOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/hetzner/sizes][%d] listHetznerSizesNoCredentialsOK  %+v", 200, o.Payload)
}

func (o *ListHetznerSizesNoCredentialsOK) GetPayload() *models.HetznerSizeList {
	return o.Payload
}

func (o *ListHetznerSizesNoCredentialsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.HetznerSizeList)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListHetznerSizesNoCredentialsDefault creates a ListHetznerSizesNoCredentialsDefault with default headers values
func NewListHetznerSizesNoCredentialsDefault(code int) *ListHetznerSizesNoCredentialsDefault {
	return &ListHetznerSizesNoCredentialsDefault{
		_statusCode: code,
	}
}

/*ListHetznerSizesNoCredentialsDefault handles this case with default header values.

errorResponse
*/
type ListHetznerSizesNoCredentialsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list hetzner sizes no credentials default response
func (o *ListHetznerSizesNoCredentialsDefault) Code() int {
	return o._statusCode
}

func (o *ListHetznerSizesNoCredentialsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/hetzner/sizes][%d] listHetznerSizesNoCredentials default  %+v", o._statusCode, o.Payload)
}

func (o *ListHetznerSizesNoCredentialsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListHetznerSizesNoCredentialsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
