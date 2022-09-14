// Code generated by go-swagger; DO NOT EDIT.

package kubevirt

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListKubeVirtVMIPresetsReader is a Reader for the ListKubeVirtVMIPresets structure.
type ListKubeVirtVMIPresetsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListKubeVirtVMIPresetsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListKubeVirtVMIPresetsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListKubeVirtVMIPresetsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListKubeVirtVMIPresetsOK creates a ListKubeVirtVMIPresetsOK with default headers values
func NewListKubeVirtVMIPresetsOK() *ListKubeVirtVMIPresetsOK {
	return &ListKubeVirtVMIPresetsOK{}
}

/*
ListKubeVirtVMIPresetsOK describes a response with status code 200, with default header values.

VirtualMachineInstancePresetList
*/
type ListKubeVirtVMIPresetsOK struct {
	Payload models.VirtualMachineInstancePresetList
}

// IsSuccess returns true when this list kube virt Vm i presets o k response has a 2xx status code
func (o *ListKubeVirtVMIPresetsOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list kube virt Vm i presets o k response has a 3xx status code
func (o *ListKubeVirtVMIPresetsOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list kube virt Vm i presets o k response has a 4xx status code
func (o *ListKubeVirtVMIPresetsOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list kube virt Vm i presets o k response has a 5xx status code
func (o *ListKubeVirtVMIPresetsOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list kube virt Vm i presets o k response a status code equal to that given
func (o *ListKubeVirtVMIPresetsOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListKubeVirtVMIPresetsOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/kubevirt/vmflavors][%d] listKubeVirtVmIPresetsOK  %+v", 200, o.Payload)
}

func (o *ListKubeVirtVMIPresetsOK) String() string {
	return fmt.Sprintf("[GET /api/v2/providers/kubevirt/vmflavors][%d] listKubeVirtVmIPresetsOK  %+v", 200, o.Payload)
}

func (o *ListKubeVirtVMIPresetsOK) GetPayload() models.VirtualMachineInstancePresetList {
	return o.Payload
}

func (o *ListKubeVirtVMIPresetsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListKubeVirtVMIPresetsDefault creates a ListKubeVirtVMIPresetsDefault with default headers values
func NewListKubeVirtVMIPresetsDefault(code int) *ListKubeVirtVMIPresetsDefault {
	return &ListKubeVirtVMIPresetsDefault{
		_statusCode: code,
	}
}

/*
ListKubeVirtVMIPresetsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListKubeVirtVMIPresetsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list kube virt VM i presets default response
func (o *ListKubeVirtVMIPresetsDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this list kube virt VM i presets default response has a 2xx status code
func (o *ListKubeVirtVMIPresetsDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this list kube virt VM i presets default response has a 3xx status code
func (o *ListKubeVirtVMIPresetsDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this list kube virt VM i presets default response has a 4xx status code
func (o *ListKubeVirtVMIPresetsDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this list kube virt VM i presets default response has a 5xx status code
func (o *ListKubeVirtVMIPresetsDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this list kube virt VM i presets default response a status code equal to that given
func (o *ListKubeVirtVMIPresetsDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *ListKubeVirtVMIPresetsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/providers/kubevirt/vmflavors][%d] listKubeVirtVMIPresets default  %+v", o._statusCode, o.Payload)
}

func (o *ListKubeVirtVMIPresetsDefault) String() string {
	return fmt.Sprintf("[GET /api/v2/providers/kubevirt/vmflavors][%d] listKubeVirtVMIPresets default  %+v", o._statusCode, o.Payload)
}

func (o *ListKubeVirtVMIPresetsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListKubeVirtVMIPresetsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
