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

// UpdateRuleAction invokes the iot.UpdateRuleAction API synchronously
// api document: https://help.aliyun.com/api/iot/updateruleaction.html
func (client *Client) UpdateRuleAction(request *UpdateRuleActionRequest) (response *UpdateRuleActionResponse, err error) {
	response = CreateUpdateRuleActionResponse()
	err = client.DoAction(request, response)
	return
}

// UpdateRuleActionWithChan invokes the iot.UpdateRuleAction API asynchronously
// api document: https://help.aliyun.com/api/iot/updateruleaction.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) UpdateRuleActionWithChan(request *UpdateRuleActionRequest) (<-chan *UpdateRuleActionResponse, <-chan error) {
	responseChan := make(chan *UpdateRuleActionResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.UpdateRuleAction(request)
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

// UpdateRuleActionWithCallback invokes the iot.UpdateRuleAction API asynchronously
// api document: https://help.aliyun.com/api/iot/updateruleaction.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) UpdateRuleActionWithCallback(request *UpdateRuleActionRequest, callback func(response *UpdateRuleActionResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *UpdateRuleActionResponse
		var err error
		defer close(result)
		response, err = client.UpdateRuleAction(request)
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

// UpdateRuleActionRequest is the request struct for api UpdateRuleAction
type UpdateRuleActionRequest struct {
	*requests.RpcRequest
	Configuration string           `position:"Query" name:"Configuration"`
	IotInstanceId string           `position:"Query" name:"IotInstanceId"`
	ActionId      requests.Integer `position:"Query" name:"ActionId"`
	Type          string           `position:"Query" name:"Type"`
}

// UpdateRuleActionResponse is the response struct for api UpdateRuleAction
type UpdateRuleActionResponse struct {
	*responses.BaseResponse
	RequestId    string `json:"RequestId" xml:"RequestId"`
	Code         string `json:"Code" xml:"Code"`
	Success      bool   `json:"Success" xml:"Success"`
	ErrorMessage string `json:"ErrorMessage" xml:"ErrorMessage"`
}

// CreateUpdateRuleActionRequest creates a request to invoke UpdateRuleAction API
func CreateUpdateRuleActionRequest() (request *UpdateRuleActionRequest) {
	request = &UpdateRuleActionRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Iot", "2018-01-20", "UpdateRuleAction", "iot", "openAPI")
	return
}

// CreateUpdateRuleActionResponse creates a response to parse from UpdateRuleAction response
func CreateUpdateRuleActionResponse() (response *UpdateRuleActionResponse) {
	response = &UpdateRuleActionResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
