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

// UnassignPrivateIpAddresses invokes the ecs.UnassignPrivateIpAddresses API synchronously
// api document: https://help.aliyun.com/api/ecs/unassignprivateipaddresses.html
func (client *Client) UnassignPrivateIpAddresses(request *UnassignPrivateIpAddressesRequest) (response *UnassignPrivateIpAddressesResponse, err error) {
	response = CreateUnassignPrivateIpAddressesResponse()
	err = client.DoAction(request, response)
	return
}

// UnassignPrivateIpAddressesWithChan invokes the ecs.UnassignPrivateIpAddresses API asynchronously
// api document: https://help.aliyun.com/api/ecs/unassignprivateipaddresses.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) UnassignPrivateIpAddressesWithChan(request *UnassignPrivateIpAddressesRequest) (<-chan *UnassignPrivateIpAddressesResponse, <-chan error) {
	responseChan := make(chan *UnassignPrivateIpAddressesResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.UnassignPrivateIpAddresses(request)
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

// UnassignPrivateIpAddressesWithCallback invokes the ecs.UnassignPrivateIpAddresses API asynchronously
// api document: https://help.aliyun.com/api/ecs/unassignprivateipaddresses.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) UnassignPrivateIpAddressesWithCallback(request *UnassignPrivateIpAddressesRequest, callback func(response *UnassignPrivateIpAddressesResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *UnassignPrivateIpAddressesResponse
		var err error
		defer close(result)
		response, err = client.UnassignPrivateIpAddresses(request)
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

// UnassignPrivateIpAddressesRequest is the request struct for api UnassignPrivateIpAddresses
type UnassignPrivateIpAddressesRequest struct {
	*requests.RpcRequest
	ResourceOwnerId      requests.Integer `position:"Query" name:"ResourceOwnerId"`
	ResourceOwnerAccount string           `position:"Query" name:"ResourceOwnerAccount"`
	OwnerAccount         string           `position:"Query" name:"OwnerAccount"`
	OwnerId              requests.Integer `position:"Query" name:"OwnerId"`
	PrivateIpAddress     *[]string        `position:"Query" name:"PrivateIpAddress"  type:"Repeated"`
	NetworkInterfaceId   string           `position:"Query" name:"NetworkInterfaceId"`
}

// UnassignPrivateIpAddressesResponse is the response struct for api UnassignPrivateIpAddresses
type UnassignPrivateIpAddressesResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreateUnassignPrivateIpAddressesRequest creates a request to invoke UnassignPrivateIpAddresses API
func CreateUnassignPrivateIpAddressesRequest() (request *UnassignPrivateIpAddressesRequest) {
	request = &UnassignPrivateIpAddressesRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Ecs", "2014-05-26", "UnassignPrivateIpAddresses", "ecs", "openAPI")
	return
}

// CreateUnassignPrivateIpAddressesResponse creates a response to parse from UnassignPrivateIpAddresses response
func CreateUnassignPrivateIpAddressesResponse() (response *UnassignPrivateIpAddressesResponse) {
	response = &UnassignPrivateIpAddressesResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
