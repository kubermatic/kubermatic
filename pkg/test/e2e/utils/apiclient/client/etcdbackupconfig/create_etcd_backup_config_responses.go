// Code generated by go-swagger; DO NOT EDIT.

package etcdbackupconfig

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// CreateEtcdBackupConfigReader is a Reader for the CreateEtcdBackupConfig structure.
type CreateEtcdBackupConfigReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *CreateEtcdBackupConfigReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 201:
		result := NewCreateEtcdBackupConfigCreated()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewCreateEtcdBackupConfigUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewCreateEtcdBackupConfigForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewCreateEtcdBackupConfigDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewCreateEtcdBackupConfigCreated creates a CreateEtcdBackupConfigCreated with default headers values
func NewCreateEtcdBackupConfigCreated() *CreateEtcdBackupConfigCreated {
	return &CreateEtcdBackupConfigCreated{}
}

/*CreateEtcdBackupConfigCreated handles this case with default header values.

EtcdBackupConfig
*/
type CreateEtcdBackupConfigCreated struct {
	Payload *models.EtcdBackupConfig
}

func (o *CreateEtcdBackupConfigCreated) Error() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/etcdbackupconfigs][%d] createEtcdBackupConfigCreated  %+v", 201, o.Payload)
}

func (o *CreateEtcdBackupConfigCreated) GetPayload() *models.EtcdBackupConfig {
	return o.Payload
}

func (o *CreateEtcdBackupConfigCreated) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.EtcdBackupConfig)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewCreateEtcdBackupConfigUnauthorized creates a CreateEtcdBackupConfigUnauthorized with default headers values
func NewCreateEtcdBackupConfigUnauthorized() *CreateEtcdBackupConfigUnauthorized {
	return &CreateEtcdBackupConfigUnauthorized{}
}

/*CreateEtcdBackupConfigUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type CreateEtcdBackupConfigUnauthorized struct {
}

func (o *CreateEtcdBackupConfigUnauthorized) Error() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/etcdbackupconfigs][%d] createEtcdBackupConfigUnauthorized ", 401)
}

func (o *CreateEtcdBackupConfigUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateEtcdBackupConfigForbidden creates a CreateEtcdBackupConfigForbidden with default headers values
func NewCreateEtcdBackupConfigForbidden() *CreateEtcdBackupConfigForbidden {
	return &CreateEtcdBackupConfigForbidden{}
}

/*CreateEtcdBackupConfigForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type CreateEtcdBackupConfigForbidden struct {
}

func (o *CreateEtcdBackupConfigForbidden) Error() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/etcdbackupconfigs][%d] createEtcdBackupConfigForbidden ", 403)
}

func (o *CreateEtcdBackupConfigForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateEtcdBackupConfigDefault creates a CreateEtcdBackupConfigDefault with default headers values
func NewCreateEtcdBackupConfigDefault(code int) *CreateEtcdBackupConfigDefault {
	return &CreateEtcdBackupConfigDefault{
		_statusCode: code,
	}
}

/*CreateEtcdBackupConfigDefault handles this case with default header values.

errorResponse
*/
type CreateEtcdBackupConfigDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the create etcd backup config default response
func (o *CreateEtcdBackupConfigDefault) Code() int {
	return o._statusCode
}

func (o *CreateEtcdBackupConfigDefault) Error() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/etcdbackupconfigs][%d] createEtcdBackupConfig default  %+v", o._statusCode, o.Payload)
}

func (o *CreateEtcdBackupConfigDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *CreateEtcdBackupConfigDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
