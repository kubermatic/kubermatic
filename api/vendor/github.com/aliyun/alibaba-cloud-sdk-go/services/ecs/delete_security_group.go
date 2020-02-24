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

// DeleteSecurityGroup invokes the ecs.DeleteSecurityGroup API synchronously
// api document: https://help.aliyun.com/api/ecs/deletesecuritygroup.html
func (client *Client) DeleteSecurityGroup(request *DeleteSecurityGroupRequest) (response *DeleteSecurityGroupResponse, err error) {
	response = CreateDeleteSecurityGroupResponse()
	err = client.DoAction(request, response)
	return
}

// DeleteSecurityGroupWithChan invokes the ecs.DeleteSecurityGroup API asynchronously
// api document: https://help.aliyun.com/api/ecs/deletesecuritygroup.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DeleteSecurityGroupWithChan(request *DeleteSecurityGroupRequest) (<-chan *DeleteSecurityGroupResponse, <-chan error) {
	responseChan := make(chan *DeleteSecurityGroupResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DeleteSecurityGroup(request)
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

// DeleteSecurityGroupWithCallback invokes the ecs.DeleteSecurityGroup API asynchronously
// api document: https://help.aliyun.com/api/ecs/deletesecuritygroup.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DeleteSecurityGroupWithCallback(request *DeleteSecurityGroupRequest, callback func(response *DeleteSecurityGroupResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DeleteSecurityGroupResponse
		var err error
		defer close(result)
		response, err = client.DeleteSecurityGroup(request)
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

// DeleteSecurityGroupRequest is the request struct for api DeleteSecurityGroup
type DeleteSecurityGroupRequest struct {
	*requests.RpcRequest
	ResourceOwnerId      requests.Integer `position:"Query" name:"ResourceOwnerId"`
	ResourceOwnerAccount string           `position:"Query" name:"ResourceOwnerAccount"`
	OwnerAccount         string           `position:"Query" name:"OwnerAccount"`
	SecurityGroupId      string           `position:"Query" name:"SecurityGroupId"`
	OwnerId              requests.Integer `position:"Query" name:"OwnerId"`
}

// DeleteSecurityGroupResponse is the response struct for api DeleteSecurityGroup
type DeleteSecurityGroupResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreateDeleteSecurityGroupRequest creates a request to invoke DeleteSecurityGroup API
func CreateDeleteSecurityGroupRequest() (request *DeleteSecurityGroupRequest) {
	request = &DeleteSecurityGroupRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Ecs", "2014-05-26", "DeleteSecurityGroup", "ecs", "openAPI")
	return
}

// CreateDeleteSecurityGroupResponse creates a response to parse from DeleteSecurityGroup response
func CreateDeleteSecurityGroupResponse() (response *DeleteSecurityGroupResponse) {
	response = &DeleteSecurityGroupResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
