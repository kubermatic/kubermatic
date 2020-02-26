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

// DeleteVirtualBorderRouter invokes the ecs.DeleteVirtualBorderRouter API synchronously
// api document: https://help.aliyun.com/api/ecs/deletevirtualborderrouter.html
func (client *Client) DeleteVirtualBorderRouter(request *DeleteVirtualBorderRouterRequest) (response *DeleteVirtualBorderRouterResponse, err error) {
	response = CreateDeleteVirtualBorderRouterResponse()
	err = client.DoAction(request, response)
	return
}

// DeleteVirtualBorderRouterWithChan invokes the ecs.DeleteVirtualBorderRouter API asynchronously
// api document: https://help.aliyun.com/api/ecs/deletevirtualborderrouter.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DeleteVirtualBorderRouterWithChan(request *DeleteVirtualBorderRouterRequest) (<-chan *DeleteVirtualBorderRouterResponse, <-chan error) {
	responseChan := make(chan *DeleteVirtualBorderRouterResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DeleteVirtualBorderRouter(request)
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

// DeleteVirtualBorderRouterWithCallback invokes the ecs.DeleteVirtualBorderRouter API asynchronously
// api document: https://help.aliyun.com/api/ecs/deletevirtualborderrouter.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DeleteVirtualBorderRouterWithCallback(request *DeleteVirtualBorderRouterRequest, callback func(response *DeleteVirtualBorderRouterResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DeleteVirtualBorderRouterResponse
		var err error
		defer close(result)
		response, err = client.DeleteVirtualBorderRouter(request)
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

// DeleteVirtualBorderRouterRequest is the request struct for api DeleteVirtualBorderRouter
type DeleteVirtualBorderRouterRequest struct {
	*requests.RpcRequest
	ResourceOwnerId      requests.Integer `position:"Query" name:"ResourceOwnerId"`
	ResourceOwnerAccount string           `position:"Query" name:"ResourceOwnerAccount"`
	ClientToken          string           `position:"Query" name:"ClientToken"`
	OwnerAccount         string           `position:"Query" name:"OwnerAccount"`
	UserCidr             string           `position:"Query" name:"UserCidr"`
	VbrId                string           `position:"Query" name:"VbrId"`
	OwnerId              requests.Integer `position:"Query" name:"OwnerId"`
}

// DeleteVirtualBorderRouterResponse is the response struct for api DeleteVirtualBorderRouter
type DeleteVirtualBorderRouterResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreateDeleteVirtualBorderRouterRequest creates a request to invoke DeleteVirtualBorderRouter API
func CreateDeleteVirtualBorderRouterRequest() (request *DeleteVirtualBorderRouterRequest) {
	request = &DeleteVirtualBorderRouterRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Ecs", "2014-05-26", "DeleteVirtualBorderRouter", "ecs", "openAPI")
	return
}

// CreateDeleteVirtualBorderRouterResponse creates a response to parse from DeleteVirtualBorderRouter response
func CreateDeleteVirtualBorderRouterResponse() (response *DeleteVirtualBorderRouterResponse) {
	response = &DeleteVirtualBorderRouterResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
