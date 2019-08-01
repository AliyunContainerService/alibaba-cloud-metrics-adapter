package smartag

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

// DescribeNetworkOptimizationSags invokes the smartag.DescribeNetworkOptimizationSags API synchronously
// api document: https://help.aliyun.com/api/smartag/describenetworkoptimizationsags.html
func (client *Client) DescribeNetworkOptimizationSags(request *DescribeNetworkOptimizationSagsRequest) (response *DescribeNetworkOptimizationSagsResponse, err error) {
	response = CreateDescribeNetworkOptimizationSagsResponse()
	err = client.DoAction(request, response)
	return
}

// DescribeNetworkOptimizationSagsWithChan invokes the smartag.DescribeNetworkOptimizationSags API asynchronously
// api document: https://help.aliyun.com/api/smartag/describenetworkoptimizationsags.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DescribeNetworkOptimizationSagsWithChan(request *DescribeNetworkOptimizationSagsRequest) (<-chan *DescribeNetworkOptimizationSagsResponse, <-chan error) {
	responseChan := make(chan *DescribeNetworkOptimizationSagsResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DescribeNetworkOptimizationSags(request)
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

// DescribeNetworkOptimizationSagsWithCallback invokes the smartag.DescribeNetworkOptimizationSags API asynchronously
// api document: https://help.aliyun.com/api/smartag/describenetworkoptimizationsags.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DescribeNetworkOptimizationSagsWithCallback(request *DescribeNetworkOptimizationSagsRequest, callback func(response *DescribeNetworkOptimizationSagsResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DescribeNetworkOptimizationSagsResponse
		var err error
		defer close(result)
		response, err = client.DescribeNetworkOptimizationSags(request)
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

// DescribeNetworkOptimizationSagsRequest is the request struct for api DescribeNetworkOptimizationSags
type DescribeNetworkOptimizationSagsRequest struct {
	*requests.RpcRequest
	ResourceOwnerId      requests.Integer `position:"Query" name:"ResourceOwnerId"`
	ResourceOwnerAccount string           `position:"Query" name:"ResourceOwnerAccount"`
	NetworkOptId         string           `position:"Query" name:"NetworkOptId"`
	PageNo               requests.Integer `position:"Query" name:"PageNo"`
	OwnerAccount         string           `position:"Query" name:"OwnerAccount"`
	PageSize             requests.Integer `position:"Query" name:"PageSize"`
	OwnerId              requests.Integer `position:"Query" name:"OwnerId"`
}

// DescribeNetworkOptimizationSagsResponse is the response struct for api DescribeNetworkOptimizationSags
type DescribeNetworkOptimizationSagsResponse struct {
	*responses.BaseResponse
	RequestId           string                                               `json:"RequestId" xml:"RequestId"`
	TotalCount          int                                                  `json:"TotalCount" xml:"TotalCount"`
	PageNo              int                                                  `json:"PageNo" xml:"PageNo"`
	PageSize            int                                                  `json:"PageSize" xml:"PageSize"`
	SmartAccessGateways SmartAccessGatewaysInDescribeNetworkOptimizationSags `json:"SmartAccessGateways" xml:"SmartAccessGateways"`
}

// CreateDescribeNetworkOptimizationSagsRequest creates a request to invoke DescribeNetworkOptimizationSags API
func CreateDescribeNetworkOptimizationSagsRequest() (request *DescribeNetworkOptimizationSagsRequest) {
	request = &DescribeNetworkOptimizationSagsRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Smartag", "2018-03-13", "DescribeNetworkOptimizationSags", "smartag", "openAPI")
	return
}

// CreateDescribeNetworkOptimizationSagsResponse creates a response to parse from DescribeNetworkOptimizationSags response
func CreateDescribeNetworkOptimizationSagsResponse() (response *DescribeNetworkOptimizationSagsResponse) {
	response = &DescribeNetworkOptimizationSagsResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
