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

// DescribeNetworkInterfaces invokes the ecs.DescribeNetworkInterfaces API synchronously
// api document: https://help.aliyun.com/api/ecs/describenetworkinterfaces.html
func (client *Client) DescribeNetworkInterfaces(request *DescribeNetworkInterfacesRequest) (response *DescribeNetworkInterfacesResponse, err error) {
	response = CreateDescribeNetworkInterfacesResponse()
	err = client.DoAction(request, response)
	return
}

// DescribeNetworkInterfacesWithChan invokes the ecs.DescribeNetworkInterfaces API asynchronously
// api document: https://help.aliyun.com/api/ecs/describenetworkinterfaces.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DescribeNetworkInterfacesWithChan(request *DescribeNetworkInterfacesRequest) (<-chan *DescribeNetworkInterfacesResponse, <-chan error) {
	responseChan := make(chan *DescribeNetworkInterfacesResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DescribeNetworkInterfaces(request)
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

// DescribeNetworkInterfacesWithCallback invokes the ecs.DescribeNetworkInterfaces API asynchronously
// api document: https://help.aliyun.com/api/ecs/describenetworkinterfaces.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DescribeNetworkInterfacesWithCallback(request *DescribeNetworkInterfacesRequest, callback func(response *DescribeNetworkInterfacesResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DescribeNetworkInterfacesResponse
		var err error
		defer close(result)
		response, err = client.DescribeNetworkInterfaces(request)
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

// DescribeNetworkInterfacesRequest is the request struct for api DescribeNetworkInterfaces
type DescribeNetworkInterfacesRequest struct {
	*requests.RpcRequest
	ResourceOwnerId      requests.Integer                `position:"Query" name:"ResourceOwnerId"`
	ServiceManaged       requests.Boolean                `position:"Query" name:"ServiceManaged"`
	SecurityGroupId      string                          `position:"Query" name:"SecurityGroupId"`
	Type                 string                          `position:"Query" name:"Type"`
	PageNumber           requests.Integer                `position:"Query" name:"PageNumber"`
	ResourceGroupId      string                          `position:"Query" name:"ResourceGroupId"`
	PageSize             requests.Integer                `position:"Query" name:"PageSize"`
	Tag                  *[]DescribeNetworkInterfacesTag `position:"Query" name:"Tag"  type:"Repeated"`
	NetworkInterfaceName string                          `position:"Query" name:"NetworkInterfaceName"`
	ResourceOwnerAccount string                          `position:"Query" name:"ResourceOwnerAccount"`
	OwnerAccount         string                          `position:"Query" name:"OwnerAccount"`
	OwnerId              requests.Integer                `position:"Query" name:"OwnerId"`
	VSwitchId            string                          `position:"Query" name:"VSwitchId"`
	PrivateIpAddress     *[]string                       `position:"Query" name:"PrivateIpAddress"  type:"Repeated"`
	InstanceId           string                          `position:"Query" name:"InstanceId"`
	VpcId                string                          `position:"Query" name:"VpcId"`
	PrimaryIpAddress     string                          `position:"Query" name:"PrimaryIpAddress"`
	NetworkInterfaceId   *[]string                       `position:"Query" name:"NetworkInterfaceId"  type:"Repeated"`
}

// DescribeNetworkInterfacesTag is a repeated param struct in DescribeNetworkInterfacesRequest
type DescribeNetworkInterfacesTag struct {
	Key   string `name:"Key"`
	Value string `name:"Value"`
}

// DescribeNetworkInterfacesResponse is the response struct for api DescribeNetworkInterfaces
type DescribeNetworkInterfacesResponse struct {
	*responses.BaseResponse
	RequestId            string               `json:"RequestId" xml:"RequestId"`
	TotalCount           int                  `json:"TotalCount" xml:"TotalCount"`
	PageNumber           int                  `json:"PageNumber" xml:"PageNumber"`
	PageSize             int                  `json:"PageSize" xml:"PageSize"`
	NetworkInterfaceSets NetworkInterfaceSets `json:"NetworkInterfaceSets" xml:"NetworkInterfaceSets"`
}

// CreateDescribeNetworkInterfacesRequest creates a request to invoke DescribeNetworkInterfaces API
func CreateDescribeNetworkInterfacesRequest() (request *DescribeNetworkInterfacesRequest) {
	request = &DescribeNetworkInterfacesRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Ecs", "2014-05-26", "DescribeNetworkInterfaces", "ecs", "openAPI")
	return
}

// CreateDescribeNetworkInterfacesResponse creates a response to parse from DescribeNetworkInterfaces response
func CreateDescribeNetworkInterfacesResponse() (response *DescribeNetworkInterfacesResponse) {
	response = &DescribeNetworkInterfacesResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
