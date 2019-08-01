package linkface

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

// DeleteDeviceAllGroup invokes the linkface.DeleteDeviceAllGroup API synchronously
// api document: https://help.aliyun.com/api/linkface/deletedeviceallgroup.html
func (client *Client) DeleteDeviceAllGroup(request *DeleteDeviceAllGroupRequest) (response *DeleteDeviceAllGroupResponse, err error) {
	response = CreateDeleteDeviceAllGroupResponse()
	err = client.DoAction(request, response)
	return
}

// DeleteDeviceAllGroupWithChan invokes the linkface.DeleteDeviceAllGroup API asynchronously
// api document: https://help.aliyun.com/api/linkface/deletedeviceallgroup.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DeleteDeviceAllGroupWithChan(request *DeleteDeviceAllGroupRequest) (<-chan *DeleteDeviceAllGroupResponse, <-chan error) {
	responseChan := make(chan *DeleteDeviceAllGroupResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DeleteDeviceAllGroup(request)
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

// DeleteDeviceAllGroupWithCallback invokes the linkface.DeleteDeviceAllGroup API asynchronously
// api document: https://help.aliyun.com/api/linkface/deletedeviceallgroup.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DeleteDeviceAllGroupWithCallback(request *DeleteDeviceAllGroupRequest, callback func(response *DeleteDeviceAllGroupResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DeleteDeviceAllGroupResponse
		var err error
		defer close(result)
		response, err = client.DeleteDeviceAllGroup(request)
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

// DeleteDeviceAllGroupRequest is the request struct for api DeleteDeviceAllGroup
type DeleteDeviceAllGroupRequest struct {
	*requests.RpcRequest
	IotId      string `position:"Body" name:"IotId"`
	DeviceName string `position:"Body" name:"DeviceName"`
	ProductKey string `position:"Body" name:"ProductKey"`
}

// DeleteDeviceAllGroupResponse is the response struct for api DeleteDeviceAllGroup
type DeleteDeviceAllGroupResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Code      int    `json:"Code" xml:"Code"`
	Message   string `json:"Message" xml:"Message"`
	Success   bool   `json:"Success" xml:"Success"`
}

// CreateDeleteDeviceAllGroupRequest creates a request to invoke DeleteDeviceAllGroup API
func CreateDeleteDeviceAllGroupRequest() (request *DeleteDeviceAllGroupRequest) {
	request = &DeleteDeviceAllGroupRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("LinkFace", "2018-07-20", "DeleteDeviceAllGroup", "", "")
	return
}

// CreateDeleteDeviceAllGroupResponse creates a response to parse from DeleteDeviceAllGroup response
func CreateDeleteDeviceAllGroupResponse() (response *DeleteDeviceAllGroupResponse) {
	response = &DeleteDeviceAllGroupResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
