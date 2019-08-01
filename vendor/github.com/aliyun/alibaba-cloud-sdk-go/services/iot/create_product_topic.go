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

// CreateProductTopic invokes the iot.CreateProductTopic API synchronously
// api document: https://help.aliyun.com/api/iot/createproducttopic.html
func (client *Client) CreateProductTopic(request *CreateProductTopicRequest) (response *CreateProductTopicResponse, err error) {
	response = CreateCreateProductTopicResponse()
	err = client.DoAction(request, response)
	return
}

// CreateProductTopicWithChan invokes the iot.CreateProductTopic API asynchronously
// api document: https://help.aliyun.com/api/iot/createproducttopic.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) CreateProductTopicWithChan(request *CreateProductTopicRequest) (<-chan *CreateProductTopicResponse, <-chan error) {
	responseChan := make(chan *CreateProductTopicResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.CreateProductTopic(request)
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

// CreateProductTopicWithCallback invokes the iot.CreateProductTopic API asynchronously
// api document: https://help.aliyun.com/api/iot/createproducttopic.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) CreateProductTopicWithCallback(request *CreateProductTopicRequest, callback func(response *CreateProductTopicResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *CreateProductTopicResponse
		var err error
		defer close(result)
		response, err = client.CreateProductTopic(request)
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

// CreateProductTopicRequest is the request struct for api CreateProductTopic
type CreateProductTopicRequest struct {
	*requests.RpcRequest
	IotInstanceId  string `position:"Query" name:"IotInstanceId"`
	ProductKey     string `position:"Query" name:"ProductKey"`
	TopicShortName string `position:"Query" name:"TopicShortName"`
	Operation      string `position:"Query" name:"Operation"`
	Desc           string `position:"Query" name:"Desc"`
}

// CreateProductTopicResponse is the response struct for api CreateProductTopic
type CreateProductTopicResponse struct {
	*responses.BaseResponse
	RequestId    string `json:"RequestId" xml:"RequestId"`
	Success      bool   `json:"Success" xml:"Success"`
	Code         string `json:"Code" xml:"Code"`
	ErrorMessage string `json:"ErrorMessage" xml:"ErrorMessage"`
	TopicId      int64  `json:"TopicId" xml:"TopicId"`
}

// CreateCreateProductTopicRequest creates a request to invoke CreateProductTopic API
func CreateCreateProductTopicRequest() (request *CreateProductTopicRequest) {
	request = &CreateProductTopicRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Iot", "2018-01-20", "CreateProductTopic", "iot", "openAPI")
	return
}

// CreateCreateProductTopicResponse creates a response to parse from CreateProductTopic response
func CreateCreateProductTopicResponse() (response *CreateProductTopicResponse) {
	response = &CreateProductTopicResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
