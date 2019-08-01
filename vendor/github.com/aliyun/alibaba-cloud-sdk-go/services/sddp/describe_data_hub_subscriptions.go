package sddp

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

// DescribeDataHubSubscriptions invokes the sddp.DescribeDataHubSubscriptions API synchronously
// api document: https://help.aliyun.com/api/sddp/describedatahubsubscriptions.html
func (client *Client) DescribeDataHubSubscriptions(request *DescribeDataHubSubscriptionsRequest) (response *DescribeDataHubSubscriptionsResponse, err error) {
	response = CreateDescribeDataHubSubscriptionsResponse()
	err = client.DoAction(request, response)
	return
}

// DescribeDataHubSubscriptionsWithChan invokes the sddp.DescribeDataHubSubscriptions API asynchronously
// api document: https://help.aliyun.com/api/sddp/describedatahubsubscriptions.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DescribeDataHubSubscriptionsWithChan(request *DescribeDataHubSubscriptionsRequest) (<-chan *DescribeDataHubSubscriptionsResponse, <-chan error) {
	responseChan := make(chan *DescribeDataHubSubscriptionsResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DescribeDataHubSubscriptions(request)
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

// DescribeDataHubSubscriptionsWithCallback invokes the sddp.DescribeDataHubSubscriptions API asynchronously
// api document: https://help.aliyun.com/api/sddp/describedatahubsubscriptions.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DescribeDataHubSubscriptionsWithCallback(request *DescribeDataHubSubscriptionsRequest, callback func(response *DescribeDataHubSubscriptionsResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DescribeDataHubSubscriptionsResponse
		var err error
		defer close(result)
		response, err = client.DescribeDataHubSubscriptions(request)
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

// DescribeDataHubSubscriptionsRequest is the request struct for api DescribeDataHubSubscriptions
type DescribeDataHubSubscriptionsRequest struct {
	*requests.RpcRequest
	TopicId     requests.Integer `position:"Query" name:"TopicId"`
	SourceIp    string           `position:"Query" name:"SourceIp"`
	FeatureType requests.Integer `position:"Query" name:"FeatureType"`
	PageSize    requests.Integer `position:"Query" name:"PageSize"`
	DepartId    requests.Integer `position:"Query" name:"DepartId"`
	CurrentPage requests.Integer `position:"Query" name:"CurrentPage"`
	Lang        string           `position:"Query" name:"Lang"`
	ProjectId   requests.Integer `position:"Query" name:"ProjectId"`
	Key         string           `position:"Query" name:"Key"`
}

// DescribeDataHubSubscriptionsResponse is the response struct for api DescribeDataHubSubscriptions
type DescribeDataHubSubscriptionsResponse struct {
	*responses.BaseResponse
	RequestId   string         `json:"RequestId" xml:"RequestId"`
	PageSize    int            `json:"PageSize" xml:"PageSize"`
	CurrentPage int            `json:"CurrentPage" xml:"CurrentPage"`
	TotalCount  int            `json:"TotalCount" xml:"TotalCount"`
	Items       []Subscription `json:"Items" xml:"Items"`
}

// CreateDescribeDataHubSubscriptionsRequest creates a request to invoke DescribeDataHubSubscriptions API
func CreateDescribeDataHubSubscriptionsRequest() (request *DescribeDataHubSubscriptionsRequest) {
	request = &DescribeDataHubSubscriptionsRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Sddp", "2019-01-03", "DescribeDataHubSubscriptions", "sddp", "openAPI")
	return
}

// CreateDescribeDataHubSubscriptionsResponse creates a response to parse from DescribeDataHubSubscriptions response
func CreateDescribeDataHubSubscriptionsResponse() (response *DescribeDataHubSubscriptionsResponse) {
	response = &DescribeDataHubSubscriptionsResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
