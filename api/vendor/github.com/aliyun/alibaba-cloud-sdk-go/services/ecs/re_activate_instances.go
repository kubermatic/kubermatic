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

// ReActivateInstances invokes the ecs.ReActivateInstances API synchronously
// api document: https://help.aliyun.com/api/ecs/reactivateinstances.html
func (client *Client) ReActivateInstances(request *ReActivateInstancesRequest) (response *ReActivateInstancesResponse, err error) {
	response = CreateReActivateInstancesResponse()
	err = client.DoAction(request, response)
	return
}

// ReActivateInstancesWithChan invokes the ecs.ReActivateInstances API asynchronously
// api document: https://help.aliyun.com/api/ecs/reactivateinstances.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) ReActivateInstancesWithChan(request *ReActivateInstancesRequest) (<-chan *ReActivateInstancesResponse, <-chan error) {
	responseChan := make(chan *ReActivateInstancesResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.ReActivateInstances(request)
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

// ReActivateInstancesWithCallback invokes the ecs.ReActivateInstances API asynchronously
// api document: https://help.aliyun.com/api/ecs/reactivateinstances.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) ReActivateInstancesWithCallback(request *ReActivateInstancesRequest, callback func(response *ReActivateInstancesResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *ReActivateInstancesResponse
		var err error
		defer close(result)
		response, err = client.ReActivateInstances(request)
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

// ReActivateInstancesRequest is the request struct for api ReActivateInstances
type ReActivateInstancesRequest struct {
	*requests.RpcRequest
	ResourceOwnerId      requests.Integer `position:"Query" name:"ResourceOwnerId"`
	InstanceId           string           `position:"Query" name:"InstanceId"`
	ResourceOwnerAccount string           `position:"Query" name:"ResourceOwnerAccount"`
	OwnerAccount         string           `position:"Query" name:"OwnerAccount"`
	OwnerId              requests.Integer `position:"Query" name:"OwnerId"`
}

// ReActivateInstancesResponse is the response struct for api ReActivateInstances
type ReActivateInstancesResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreateReActivateInstancesRequest creates a request to invoke ReActivateInstances API
func CreateReActivateInstancesRequest() (request *ReActivateInstancesRequest) {
	request = &ReActivateInstancesRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Ecs", "2014-05-26", "ReActivateInstances", "ecs", "openAPI")
	return
}

// CreateReActivateInstancesResponse creates a response to parse from ReActivateInstances response
func CreateReActivateInstancesResponse() (response *ReActivateInstancesResponse) {
	response = &ReActivateInstancesResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
