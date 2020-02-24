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

// DeleteNatGateway invokes the ecs.DeleteNatGateway API synchronously
// api document: https://help.aliyun.com/api/ecs/deletenatgateway.html
func (client *Client) DeleteNatGateway(request *DeleteNatGatewayRequest) (response *DeleteNatGatewayResponse, err error) {
	response = CreateDeleteNatGatewayResponse()
	err = client.DoAction(request, response)
	return
}

// DeleteNatGatewayWithChan invokes the ecs.DeleteNatGateway API asynchronously
// api document: https://help.aliyun.com/api/ecs/deletenatgateway.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DeleteNatGatewayWithChan(request *DeleteNatGatewayRequest) (<-chan *DeleteNatGatewayResponse, <-chan error) {
	responseChan := make(chan *DeleteNatGatewayResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DeleteNatGateway(request)
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

// DeleteNatGatewayWithCallback invokes the ecs.DeleteNatGateway API asynchronously
// api document: https://help.aliyun.com/api/ecs/deletenatgateway.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DeleteNatGatewayWithCallback(request *DeleteNatGatewayRequest, callback func(response *DeleteNatGatewayResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DeleteNatGatewayResponse
		var err error
		defer close(result)
		response, err = client.DeleteNatGateway(request)
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

// DeleteNatGatewayRequest is the request struct for api DeleteNatGateway
type DeleteNatGatewayRequest struct {
	*requests.RpcRequest
	ResourceOwnerId      requests.Integer `position:"Query" name:"ResourceOwnerId"`
	ResourceOwnerAccount string           `position:"Query" name:"ResourceOwnerAccount"`
	OwnerAccount         string           `position:"Query" name:"OwnerAccount"`
	NatGatewayId         string           `position:"Query" name:"NatGatewayId"`
	OwnerId              requests.Integer `position:"Query" name:"OwnerId"`
}

// DeleteNatGatewayResponse is the response struct for api DeleteNatGateway
type DeleteNatGatewayResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreateDeleteNatGatewayRequest creates a request to invoke DeleteNatGateway API
func CreateDeleteNatGatewayRequest() (request *DeleteNatGatewayRequest) {
	request = &DeleteNatGatewayRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Ecs", "2014-05-26", "DeleteNatGateway", "ecs", "openAPI")
	return
}

// CreateDeleteNatGatewayResponse creates a response to parse from DeleteNatGateway response
func CreateDeleteNatGatewayResponse() (response *DeleteNatGatewayResponse) {
	response = &DeleteNatGatewayResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
