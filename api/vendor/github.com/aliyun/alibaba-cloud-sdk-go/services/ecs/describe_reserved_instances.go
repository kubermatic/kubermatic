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

// DescribeReservedInstances invokes the ecs.DescribeReservedInstances API synchronously
// api document: https://help.aliyun.com/api/ecs/describereservedinstances.html
func (client *Client) DescribeReservedInstances(request *DescribeReservedInstancesRequest) (response *DescribeReservedInstancesResponse, err error) {
	response = CreateDescribeReservedInstancesResponse()
	err = client.DoAction(request, response)
	return
}

// DescribeReservedInstancesWithChan invokes the ecs.DescribeReservedInstances API asynchronously
// api document: https://help.aliyun.com/api/ecs/describereservedinstances.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DescribeReservedInstancesWithChan(request *DescribeReservedInstancesRequest) (<-chan *DescribeReservedInstancesResponse, <-chan error) {
	responseChan := make(chan *DescribeReservedInstancesResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DescribeReservedInstances(request)
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

// DescribeReservedInstancesWithCallback invokes the ecs.DescribeReservedInstances API asynchronously
// api document: https://help.aliyun.com/api/ecs/describereservedinstances.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DescribeReservedInstancesWithCallback(request *DescribeReservedInstancesRequest, callback func(response *DescribeReservedInstancesResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DescribeReservedInstancesResponse
		var err error
		defer close(result)
		response, err = client.DescribeReservedInstances(request)
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

// DescribeReservedInstancesRequest is the request struct for api DescribeReservedInstances
type DescribeReservedInstancesRequest struct {
	*requests.RpcRequest
	ResourceOwnerId      requests.Integer `position:"Query" name:"ResourceOwnerId"`
	PageNumber           requests.Integer `position:"Query" name:"PageNumber"`
	LockReason           string           `position:"Query" name:"LockReason"`
	Scope                string           `position:"Query" name:"Scope"`
	PageSize             requests.Integer `position:"Query" name:"PageSize"`
	InstanceType         string           `position:"Query" name:"InstanceType"`
	ResourceOwnerAccount string           `position:"Query" name:"ResourceOwnerAccount"`
	OwnerAccount         string           `position:"Query" name:"OwnerAccount"`
	InstanceTypeFamily   string           `position:"Query" name:"InstanceTypeFamily"`
	OwnerId              requests.Integer `position:"Query" name:"OwnerId"`
	ReservedInstanceId   *[]string        `position:"Query" name:"ReservedInstanceId"  type:"Repeated"`
	OfferingType         string           `position:"Query" name:"OfferingType"`
	ZoneId               string           `position:"Query" name:"ZoneId"`
	ReservedInstanceName string           `position:"Query" name:"ReservedInstanceName"`
	Status               *[]string        `position:"Query" name:"Status"  type:"Repeated"`
}

// DescribeReservedInstancesResponse is the response struct for api DescribeReservedInstances
type DescribeReservedInstancesResponse struct {
	*responses.BaseResponse
	RequestId         string            `json:"RequestId" xml:"RequestId"`
	TotalCount        int               `json:"TotalCount" xml:"TotalCount"`
	PageNumber        int               `json:"PageNumber" xml:"PageNumber"`
	PageSize          int               `json:"PageSize" xml:"PageSize"`
	ReservedInstances ReservedInstances `json:"ReservedInstances" xml:"ReservedInstances"`
}

// CreateDescribeReservedInstancesRequest creates a request to invoke DescribeReservedInstances API
func CreateDescribeReservedInstancesRequest() (request *DescribeReservedInstancesRequest) {
	request = &DescribeReservedInstancesRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Ecs", "2014-05-26", "DescribeReservedInstances", "ecs", "openAPI")
	return
}

// CreateDescribeReservedInstancesResponse creates a response to parse from DescribeReservedInstances response
func CreateDescribeReservedInstancesResponse() (response *DescribeReservedInstancesResponse) {
	response = &DescribeReservedInstancesResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
