package iot

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

// CreateLoRaNodesTask invokes the iot.CreateLoRaNodesTask API synchronously
// api document: https://help.aliyun.com/api/iot/createloranodestask.html
func (client *Client) CreateLoRaNodesTask(request *CreateLoRaNodesTaskRequest) (response *CreateLoRaNodesTaskResponse, err error) {
	response = CreateCreateLoRaNodesTaskResponse()
	err = client.DoAction(request, response)
	return
}

// CreateLoRaNodesTaskWithChan invokes the iot.CreateLoRaNodesTask API asynchronously
// api document: https://help.aliyun.com/api/iot/createloranodestask.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) CreateLoRaNodesTaskWithChan(request *CreateLoRaNodesTaskRequest) (<-chan *CreateLoRaNodesTaskResponse, <-chan error) {
	responseChan := make(chan *CreateLoRaNodesTaskResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.CreateLoRaNodesTask(request)
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

// CreateLoRaNodesTaskWithCallback invokes the iot.CreateLoRaNodesTask API asynchronously
// api document: https://help.aliyun.com/api/iot/createloranodestask.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) CreateLoRaNodesTaskWithCallback(request *CreateLoRaNodesTaskRequest, callback func(response *CreateLoRaNodesTaskResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *CreateLoRaNodesTaskResponse
		var err error
		defer close(result)
		response, err = client.CreateLoRaNodesTask(request)
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

// CreateLoRaNodesTaskRequest is the request struct for api CreateLoRaNodesTask
type CreateLoRaNodesTaskRequest struct {
	*requests.RpcRequest
	IotInstanceId string                           `position:"Query" name:"IotInstanceId"`
	ProductKey    string                           `position:"Query" name:"ProductKey"`
	DeviceInfo    *[]CreateLoRaNodesTaskDeviceInfo `position:"Query" name:"DeviceInfo"  type:"Repeated"`
}

// CreateLoRaNodesTaskDeviceInfo is a repeated param struct in CreateLoRaNodesTaskRequest
type CreateLoRaNodesTaskDeviceInfo struct {
	PinCode string `name:"PinCode"`
	DevEui  string `name:"DevEui"`
}

// CreateLoRaNodesTaskResponse is the response struct for api CreateLoRaNodesTask
type CreateLoRaNodesTaskResponse struct {
	*responses.BaseResponse
	RequestId    string `json:"RequestId" xml:"RequestId"`
	Success      bool   `json:"Success" xml:"Success"`
	Code         string `json:"Code" xml:"Code"`
	ErrorMessage string `json:"ErrorMessage" xml:"ErrorMessage"`
	TaskId       string `json:"TaskId" xml:"TaskId"`
}

// CreateCreateLoRaNodesTaskRequest creates a request to invoke CreateLoRaNodesTask API
func CreateCreateLoRaNodesTaskRequest() (request *CreateLoRaNodesTaskRequest) {
	request = &CreateLoRaNodesTaskRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Iot", "2018-01-20", "CreateLoRaNodesTask", "iot", "openAPI")
	return
}

// CreateCreateLoRaNodesTaskResponse creates a response to parse from CreateLoRaNodesTask response
func CreateCreateLoRaNodesTaskResponse() (response *CreateLoRaNodesTaskResponse) {
	response = &CreateLoRaNodesTaskResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
