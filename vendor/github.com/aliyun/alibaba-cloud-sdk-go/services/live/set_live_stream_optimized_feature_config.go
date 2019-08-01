package live

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

// SetLiveStreamOptimizedFeatureConfig invokes the live.SetLiveStreamOptimizedFeatureConfig API synchronously
// api document: https://help.aliyun.com/api/live/setlivestreamoptimizedfeatureconfig.html
func (client *Client) SetLiveStreamOptimizedFeatureConfig(request *SetLiveStreamOptimizedFeatureConfigRequest) (response *SetLiveStreamOptimizedFeatureConfigResponse, err error) {
	response = CreateSetLiveStreamOptimizedFeatureConfigResponse()
	err = client.DoAction(request, response)
	return
}

// SetLiveStreamOptimizedFeatureConfigWithChan invokes the live.SetLiveStreamOptimizedFeatureConfig API asynchronously
// api document: https://help.aliyun.com/api/live/setlivestreamoptimizedfeatureconfig.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) SetLiveStreamOptimizedFeatureConfigWithChan(request *SetLiveStreamOptimizedFeatureConfigRequest) (<-chan *SetLiveStreamOptimizedFeatureConfigResponse, <-chan error) {
	responseChan := make(chan *SetLiveStreamOptimizedFeatureConfigResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.SetLiveStreamOptimizedFeatureConfig(request)
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

// SetLiveStreamOptimizedFeatureConfigWithCallback invokes the live.SetLiveStreamOptimizedFeatureConfig API asynchronously
// api document: https://help.aliyun.com/api/live/setlivestreamoptimizedfeatureconfig.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) SetLiveStreamOptimizedFeatureConfigWithCallback(request *SetLiveStreamOptimizedFeatureConfigRequest, callback func(response *SetLiveStreamOptimizedFeatureConfigResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *SetLiveStreamOptimizedFeatureConfigResponse
		var err error
		defer close(result)
		response, err = client.SetLiveStreamOptimizedFeatureConfig(request)
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

// SetLiveStreamOptimizedFeatureConfigRequest is the request struct for api SetLiveStreamOptimizedFeatureConfig
type SetLiveStreamOptimizedFeatureConfigRequest struct {
	*requests.RpcRequest
	ConfigStatus string           `position:"Query" name:"ConfigStatus"`
	ConfigName   string           `position:"Query" name:"ConfigName"`
	DomainName   string           `position:"Query" name:"DomainName"`
	ConfigValue  string           `position:"Query" name:"ConfigValue"`
	OwnerId      requests.Integer `position:"Query" name:"OwnerId"`
}

// SetLiveStreamOptimizedFeatureConfigResponse is the response struct for api SetLiveStreamOptimizedFeatureConfig
type SetLiveStreamOptimizedFeatureConfigResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreateSetLiveStreamOptimizedFeatureConfigRequest creates a request to invoke SetLiveStreamOptimizedFeatureConfig API
func CreateSetLiveStreamOptimizedFeatureConfigRequest() (request *SetLiveStreamOptimizedFeatureConfigRequest) {
	request = &SetLiveStreamOptimizedFeatureConfigRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("live", "2016-11-01", "SetLiveStreamOptimizedFeatureConfig", "live", "openAPI")
	return
}

// CreateSetLiveStreamOptimizedFeatureConfigResponse creates a response to parse from SetLiveStreamOptimizedFeatureConfig response
func CreateSetLiveStreamOptimizedFeatureConfigResponse() (response *SetLiveStreamOptimizedFeatureConfigResponse) {
	response = &SetLiveStreamOptimizedFeatureConfigResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
