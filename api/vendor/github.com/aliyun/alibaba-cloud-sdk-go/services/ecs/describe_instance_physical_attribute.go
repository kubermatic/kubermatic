package ecs

//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
// Code generated by Alibaba Cloud SDK Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
)

// DescribeInstancePhysicalAttribute invokes the ecs.DescribeInstancePhysicalAttribute API synchronously
// api document: https://help.aliyun.com/api/ecs/describeinstancephysicalattribute.html
func (client *Client) DescribeInstancePhysicalAttribute(request *DescribeInstancePhysicalAttributeRequest) (response *DescribeInstancePhysicalAttributeResponse, err error) {
	response = CreateDescribeInstancePhysicalAttributeResponse()
	err = client.DoAction(request, response)
	return
}

// DescribeInstancePhysicalAttributeWithChan invokes the ecs.DescribeInstancePhysicalAttribute API asynchronously
// api document: https://help.aliyun.com/api/ecs/describeinstancephysicalattribute.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DescribeInstancePhysicalAttributeWithChan(request *DescribeInstancePhysicalAttributeRequest) (<-chan *DescribeInstancePhysicalAttributeResponse, <-chan error) {
	responseChan := make(chan *DescribeInstancePhysicalAttributeResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DescribeInstancePhysicalAttribute(request)
		if err != nil {
			errChan <- err
		} else {
			responseChan <- response
		}
	})
	if err != nil {
		errChan <- err
		close(responseChan)
		close(errChan)
	}
	return responseChan, errChan
}

// DescribeInstancePhysicalAttributeWithCallback invokes the ecs.DescribeInstancePhysicalAttribute API asynchronously
// api document: https://help.aliyun.com/api/ecs/describeinstancephysicalattribute.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DescribeInstancePhysicalAttributeWithCallback(request *DescribeInstancePhysicalAttributeRequest, callback func(response *DescribeInstancePhysicalAttributeResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DescribeInstancePhysicalAttributeResponse
		var err error
		defer close(result)
		response, err = client.DescribeInstancePhysicalAttribute(request)
		callback(response, err)
		result <- 1
	})
	if err != nil {
		defer close(result)
		callback(nil, err)
		result <- 0
	}
	return result
}

// DescribeInstancePhysicalAttributeRequest is the request struct for api DescribeInstancePhysicalAttribute
type DescribeInstancePhysicalAttributeRequest struct {
	*requests.RpcRequest
	ResourceOwnerId      requests.Integer `position:"Query" name:"ResourceOwnerId"`
	InstanceId           string           `position:"Query" name:"InstanceId"`
	ResourceOwnerAccount string           `position:"Query" name:"ResourceOwnerAccount"`
	OwnerAccount         string           `position:"Query" name:"OwnerAccount"`
	OwnerId              requests.Integer `position:"Query" name:"OwnerId"`
}

// DescribeInstancePhysicalAttributeResponse is the response struct for api DescribeInstancePhysicalAttribute
type DescribeInstancePhysicalAttributeResponse struct {
	*responses.BaseResponse
	RequestId        string `json:"RequestId" xml:"RequestId"`
	InstanceId       string `json:"InstanceId" xml:"InstanceId"`
	VlanId           string `json:"VlanId" xml:"VlanId"`
	NodeControllerId string `json:"NodeControllerId" xml:"NodeControllerId"`
	RackId           string `json:"RackId" xml:"RackId"`
}

// CreateDescribeInstancePhysicalAttributeRequest creates a request to invoke DescribeInstancePhysicalAttribute API
func CreateDescribeInstancePhysicalAttributeRequest() (request *DescribeInstancePhysicalAttributeRequest) {
	request = &DescribeInstancePhysicalAttributeRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Ecs", "2014-05-26", "DescribeInstancePhysicalAttribute", "ecs", "openAPI")
	return
}

// CreateDescribeInstancePhysicalAttributeResponse creates a response to parse from DescribeInstancePhysicalAttribute response
func CreateDescribeInstancePhysicalAttributeResponse() (response *DescribeInstancePhysicalAttributeResponse) {
	response = &DescribeInstancePhysicalAttributeResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
