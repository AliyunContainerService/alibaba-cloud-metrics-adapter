package cloudphoto

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

// DeleteEvent invokes the cloudphoto.DeleteEvent API synchronously
// api document: https://help.aliyun.com/api/cloudphoto/deleteevent.html
func (client *Client) DeleteEvent(request *DeleteEventRequest) (response *DeleteEventResponse, err error) {
	response = CreateDeleteEventResponse()
	err = client.DoAction(request, response)
	return
}

// DeleteEventWithChan invokes the cloudphoto.DeleteEvent API asynchronously
// api document: https://help.aliyun.com/api/cloudphoto/deleteevent.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DeleteEventWithChan(request *DeleteEventRequest) (<-chan *DeleteEventResponse, <-chan error) {
	responseChan := make(chan *DeleteEventResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DeleteEvent(request)
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

// DeleteEventWithCallback invokes the cloudphoto.DeleteEvent API asynchronously
// api document: https://help.aliyun.com/api/cloudphoto/deleteevent.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DeleteEventWithCallback(request *DeleteEventRequest, callback func(response *DeleteEventResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DeleteEventResponse
		var err error
		defer close(result)
		response, err = client.DeleteEvent(request)
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

// DeleteEventRequest is the request struct for api DeleteEvent
type DeleteEventRequest struct {
	*requests.RpcRequest
	EventId   requests.Integer `position:"Query" name:"EventId"`
	LibraryId string           `position:"Query" name:"LibraryId"`
	StoreName string           `position:"Query" name:"StoreName"`
}

// DeleteEventResponse is the response struct for api DeleteEvent
type DeleteEventResponse struct {
	*responses.BaseResponse
	Code      string `json:"Code" xml:"Code"`
	Message   string `json:"Message" xml:"Message"`
	RequestId string `json:"RequestId" xml:"RequestId"`
	Action    string `json:"Action" xml:"Action"`
}

// CreateDeleteEventRequest creates a request to invoke DeleteEvent API
func CreateDeleteEventRequest() (request *DeleteEventRequest) {
	request = &DeleteEventRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("CloudPhoto", "2017-07-11", "DeleteEvent", "cloudphoto", "openAPI")
	return
}

// CreateDeleteEventResponse creates a response to parse from DeleteEvent response
func CreateDeleteEventResponse() (response *DeleteEventResponse) {
	response = &DeleteEventResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
