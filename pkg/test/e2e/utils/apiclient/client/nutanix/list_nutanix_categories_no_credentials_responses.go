// Code generated by go-swagger; DO NOT EDIT.

package nutanix

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListNutanixCategoriesNoCredentialsReader is a Reader for the ListNutanixCategoriesNoCredentials structure.
type ListNutanixCategoriesNoCredentialsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListNutanixCategoriesNoCredentialsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListNutanixCategoriesNoCredentialsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListNutanixCategoriesNoCredentialsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListNutanixCategoriesNoCredentialsOK creates a ListNutanixCategoriesNoCredentialsOK with default headers values
func NewListNutanixCategoriesNoCredentialsOK() *ListNutanixCategoriesNoCredentialsOK {
	return &ListNutanixCategoriesNoCredentialsOK{}
}

/* ListNutanixCategoriesNoCredentialsOK describes a response with status code 200, with default header values.

NutanixCategoryList
*/
type ListNutanixCategoriesNoCredentialsOK struct {
	Payload models.NutanixCategoryList
}

func (o *ListNutanixCategoriesNoCredentialsOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/nutanix/categories][%d] listNutanixCategoriesNoCredentialsOK  %+v", 200, o.Payload)
}
func (o *ListNutanixCategoriesNoCredentialsOK) GetPayload() models.NutanixCategoryList {
	return o.Payload
}

func (o *ListNutanixCategoriesNoCredentialsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListNutanixCategoriesNoCredentialsDefault creates a ListNutanixCategoriesNoCredentialsDefault with default headers values
func NewListNutanixCategoriesNoCredentialsDefault(code int) *ListNutanixCategoriesNoCredentialsDefault {
	return &ListNutanixCategoriesNoCredentialsDefault{
		_statusCode: code,
	}
}

/* ListNutanixCategoriesNoCredentialsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListNutanixCategoriesNoCredentialsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list nutanix categories no credentials default response
func (o *ListNutanixCategoriesNoCredentialsDefault) Code() int {
	return o._statusCode
}

func (o *ListNutanixCategoriesNoCredentialsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/nutanix/categories][%d] listNutanixCategoriesNoCredentials default  %+v", o._statusCode, o.Payload)
}
func (o *ListNutanixCategoriesNoCredentialsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListNutanixCategoriesNoCredentialsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
