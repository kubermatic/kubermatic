// Code generated by go-swagger; DO NOT EDIT.

package aws

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// ListAWSSizesNoCredentialsReader is a Reader for the ListAWSSizesNoCredentials structure.
type ListAWSSizesNoCredentialsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListAWSSizesNoCredentialsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListAWSSizesNoCredentialsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListAWSSizesNoCredentialsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListAWSSizesNoCredentialsOK creates a ListAWSSizesNoCredentialsOK with default headers values
func NewListAWSSizesNoCredentialsOK() *ListAWSSizesNoCredentialsOK {
	return &ListAWSSizesNoCredentialsOK{}
}

/*ListAWSSizesNoCredentialsOK handles this case with default header values.

AWSSizeList
*/
type ListAWSSizesNoCredentialsOK struct {
	Payload models.AWSSizeList
}

func (o *ListAWSSizesNoCredentialsOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/aws/sizes][%d] listAWSSizesNoCredentialsOK  %+v", 200, o.Payload)
}

func (o *ListAWSSizesNoCredentialsOK) GetPayload() models.AWSSizeList {
	return o.Payload
}

func (o *ListAWSSizesNoCredentialsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListAWSSizesNoCredentialsDefault creates a ListAWSSizesNoCredentialsDefault with default headers values
func NewListAWSSizesNoCredentialsDefault(code int) *ListAWSSizesNoCredentialsDefault {
	return &ListAWSSizesNoCredentialsDefault{
		_statusCode: code,
	}
}

/*ListAWSSizesNoCredentialsDefault handles this case with default header values.

errorResponse
*/
type ListAWSSizesNoCredentialsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list a w s sizes no credentials default response
func (o *ListAWSSizesNoCredentialsDefault) Code() int {
	return o._statusCode
}

func (o *ListAWSSizesNoCredentialsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/aws/sizes][%d] listAWSSizesNoCredentials default  %+v", o._statusCode, o.Payload)
}

func (o *ListAWSSizesNoCredentialsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListAWSSizesNoCredentialsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
