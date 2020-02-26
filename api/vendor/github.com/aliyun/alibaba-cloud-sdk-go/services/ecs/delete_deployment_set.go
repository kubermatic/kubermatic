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

// DeleteDeploymentSet invokes the ecs.DeleteDeploymentSet API synchronously
// api document: https://help.aliyun.com/api/ecs/deletedeploymentset.html
func (client *Client) DeleteDeploymentSet(request *DeleteDeploymentSetRequest) (response *DeleteDeploymentSetResponse, err error) {
	response = CreateDeleteDeploymentSetResponse()
	err = client.DoAction(request, response)
	return
}

// DeleteDeploymentSetWithChan invokes the ecs.DeleteDeploymentSet API asynchronously
// api document: https://help.aliyun.com/api/ecs/deletedeploymentset.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DeleteDeploymentSetWithChan(request *DeleteDeploymentSetRequest) (<-chan *DeleteDeploymentSetResponse, <-chan error) {
	responseChan := make(chan *DeleteDeploymentSetResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DeleteDeploymentSet(request)
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

// DeleteDeploymentSetWithCallback invokes the ecs.DeleteDeploymentSet API asynchronously
// api document: https://help.aliyun.com/api/ecs/deletedeploymentset.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DeleteDeploymentSetWithCallback(request *DeleteDeploymentSetRequest, callback func(response *DeleteDeploymentSetResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DeleteDeploymentSetResponse
		var err error
		defer close(result)
		response, err = client.DeleteDeploymentSet(request)
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

// DeleteDeploymentSetRequest is the request struct for api DeleteDeploymentSet
type DeleteDeploymentSetRequest struct {
	*requests.RpcRequest
	DeploymentSetId      string           `position:"Query" name:"DeploymentSetId"`
	ResourceOwnerId      requests.Integer `position:"Query" name:"ResourceOwnerId"`
	ResourceOwnerAccount string           `position:"Query" name:"ResourceOwnerAccount"`
	OwnerAccount         string           `position:"Query" name:"OwnerAccount"`
	OwnerId              requests.Integer `position:"Query" name:"OwnerId"`
}

// DeleteDeploymentSetResponse is the response struct for api DeleteDeploymentSet
type DeleteDeploymentSetResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreateDeleteDeploymentSetRequest creates a request to invoke DeleteDeploymentSet API
func CreateDeleteDeploymentSetRequest() (request *DeleteDeploymentSetRequest) {
	request = &DeleteDeploymentSetRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Ecs", "2014-05-26", "DeleteDeploymentSet", "ecs", "openAPI")
	return
}

// CreateDeleteDeploymentSetResponse creates a response to parse from DeleteDeploymentSet response
func CreateDeleteDeploymentSetResponse() (response *DeleteDeploymentSetResponse) {
	response = &DeleteDeploymentSetResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
