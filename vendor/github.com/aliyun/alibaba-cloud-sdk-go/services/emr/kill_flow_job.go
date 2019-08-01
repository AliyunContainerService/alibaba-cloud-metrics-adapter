package emr

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

// KillFlowJob invokes the emr.KillFlowJob API synchronously
// api document: https://help.aliyun.com/api/emr/killflowjob.html
func (client *Client) KillFlowJob(request *KillFlowJobRequest) (response *KillFlowJobResponse, err error) {
	response = CreateKillFlowJobResponse()
	err = client.DoAction(request, response)
	return
}

// KillFlowJobWithChan invokes the emr.KillFlowJob API asynchronously
// api document: https://help.aliyun.com/api/emr/killflowjob.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) KillFlowJobWithChan(request *KillFlowJobRequest) (<-chan *KillFlowJobResponse, <-chan error) {
	responseChan := make(chan *KillFlowJobResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.KillFlowJob(request)
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

// KillFlowJobWithCallback invokes the emr.KillFlowJob API asynchronously
// api document: https://help.aliyun.com/api/emr/killflowjob.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) KillFlowJobWithCallback(request *KillFlowJobRequest, callback func(response *KillFlowJobResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *KillFlowJobResponse
		var err error
		defer close(result)
		response, err = client.KillFlowJob(request)
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

// KillFlowJobRequest is the request struct for api KillFlowJob
type KillFlowJobRequest struct {
	*requests.RpcRequest
	JobInstanceId string `position:"Query" name:"JobInstanceId"`
	ProjectId     string `position:"Query" name:"ProjectId"`
}

// KillFlowJobResponse is the response struct for api KillFlowJob
type KillFlowJobResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Data      bool   `json:"Data" xml:"Data"`
}

// CreateKillFlowJobRequest creates a request to invoke KillFlowJob API
func CreateKillFlowJobRequest() (request *KillFlowJobRequest) {
	request = &KillFlowJobRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Emr", "2016-04-08", "KillFlowJob", "emr", "openAPI")
	return
}

// CreateKillFlowJobResponse creates a response to parse from KillFlowJob response
func CreateKillFlowJobResponse() (response *KillFlowJobResponse) {
	response = &KillFlowJobResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
