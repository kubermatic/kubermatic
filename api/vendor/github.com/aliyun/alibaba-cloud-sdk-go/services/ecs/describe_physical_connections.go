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

// DescribePhysicalConnections invokes the ecs.DescribePhysicalConnections API synchronously
// api document: https://help.aliyun.com/api/ecs/describephysicalconnections.html
func (client *Client) DescribePhysicalConnections(request *DescribePhysicalConnectionsRequest) (response *DescribePhysicalConnectionsResponse, err error) {
	response = CreateDescribePhysicalConnectionsResponse()
	err = client.DoAction(request, response)
	return
}

// DescribePhysicalConnectionsWithChan invokes the ecs.DescribePhysicalConnections API asynchronously
// api document: https://help.aliyun.com/api/ecs/describephysicalconnections.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DescribePhysicalConnectionsWithChan(request *DescribePhysicalConnectionsRequest) (<-chan *DescribePhysicalConnectionsResponse, <-chan error) {
	responseChan := make(chan *DescribePhysicalConnectionsResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DescribePhysicalConnections(request)
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

// DescribePhysicalConnectionsWithCallback invokes the ecs.DescribePhysicalConnections API asynchronously
// api document: https://help.aliyun.com/api/ecs/describephysicalconnections.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DescribePhysicalConnectionsWithCallback(request *DescribePhysicalConnectionsRequest, callback func(response *DescribePhysicalConnectionsResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DescribePhysicalConnectionsResponse
		var err error
		defer close(result)
		response, err = client.DescribePhysicalConnections(request)
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

// DescribePhysicalConnectionsRequest is the request struct for api DescribePhysicalConnections
type DescribePhysicalConnectionsRequest struct {
	*requests.RpcRequest
	Filter               *[]DescribePhysicalConnectionsFilter `position:"Query" name:"Filter"  type:"Repeated"`
	ResourceOwnerId      requests.Integer                     `position:"Query" name:"ResourceOwnerId"`
	ResourceOwnerAccount string                               `position:"Query" name:"ResourceOwnerAccount"`
	ClientToken          string                               `position:"Query" name:"ClientToken"`
	OwnerAccount         string                               `position:"Query" name:"OwnerAccount"`
	PageSize             requests.Integer                     `position:"Query" name:"PageSize"`
	UserCidr             string                               `position:"Query" name:"UserCidr"`
	OwnerId              requests.Integer                     `position:"Query" name:"OwnerId"`
	PageNumber           requests.Integer                     `position:"Query" name:"PageNumber"`
}

// DescribePhysicalConnectionsFilter is a repeated param struct in DescribePhysicalConnectionsRequest
type DescribePhysicalConnectionsFilter struct {
	Value *[]string `name:"Value" type:"Repeated"`
	Key   string    `name:"Key"`
}

// DescribePhysicalConnectionsResponse is the response struct for api DescribePhysicalConnections
type DescribePhysicalConnectionsResponse struct {
	*responses.BaseResponse
	RequestId             string                `json:"RequestId" xml:"RequestId"`
	PageNumber            int                   `json:"PageNumber" xml:"PageNumber"`
	PageSize              int                   `json:"PageSize" xml:"PageSize"`
	TotalCount            int                   `json:"TotalCount" xml:"TotalCount"`
	PhysicalConnectionSet PhysicalConnectionSet `json:"PhysicalConnectionSet" xml:"PhysicalConnectionSet"`
}

// CreateDescribePhysicalConnectionsRequest creates a request to invoke DescribePhysicalConnections API
func CreateDescribePhysicalConnectionsRequest() (request *DescribePhysicalConnectionsRequest) {
	request = &DescribePhysicalConnectionsRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Ecs", "2014-05-26", "DescribePhysicalConnections", "ecs", "openAPI")
	return
}

// CreateDescribePhysicalConnectionsResponse creates a response to parse from DescribePhysicalConnections response
func CreateDescribePhysicalConnectionsResponse() (response *DescribePhysicalConnectionsResponse) {
	response = &DescribePhysicalConnectionsResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
