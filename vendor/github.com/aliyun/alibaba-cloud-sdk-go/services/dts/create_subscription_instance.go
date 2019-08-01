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

package dts

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
)

// CreateSubscriptionInstance invokes the dts.CreateSubscriptionInstance API synchronously
// api document: https://help.aliyun.com/api/dts/createsubscriptioninstance.html
func (client *Client) CreateSubscriptionInstance(request *CreateSubscriptionInstanceRequest) (response *CreateSubscriptionInstanceResponse, err error) {
	response = CreateCreateSubscriptionInstanceResponse()
	err = client.DoAction(request, response)
	return
}

// CreateSubscriptionInstanceWithChan invokes the dts.CreateSubscriptionInstance API asynchronously
// api document: https://help.aliyun.com/api/dts/createsubscriptioninstance.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) CreateSubscriptionInstanceWithChan(request *CreateSubscriptionInstanceRequest) (<-chan *CreateSubscriptionInstanceResponse, <-chan error) {
	responseChan := make(chan *CreateSubscriptionInstanceResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.CreateSubscriptionInstance(request)
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

// CreateSubscriptionInstanceWithCallback invokes the dts.CreateSubscriptionInstance API asynchronously
// api document: https://help.aliyun.com/api/dts/createsubscriptioninstance.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) CreateSubscriptionInstanceWithCallback(request *CreateSubscriptionInstanceRequest, callback func(response *CreateSubscriptionInstanceResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *CreateSubscriptionInstanceResponse
		var err error
		defer close(result)
		response, err = client.CreateSubscriptionInstance(request)
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

// CreateSubscriptionInstanceRequest is the request struct for api CreateSubscriptionInstance
type CreateSubscriptionInstanceRequest struct {
	*requests.RpcRequest
	Region      string           `position:"Query" name:"Region"`
	PayType     string           `position:"Query" name:"PayType"`
	Period      string           `position:"Query" name:"Period"`
	UsedTime    requests.Integer `position:"Query" name:"UsedTime"`
	ClientToken string           `position:"Query" name:"ClientToken"`
	OwnerId     string           `position:"Query" name:"OwnerId"`
}

// CreateSubscriptionInstanceResponse is the response struct for api CreateSubscriptionInstance
type CreateSubscriptionInstanceResponse struct {
	*responses.BaseResponse
	Success                string `json:"Success" xml:"Success"`
	ErrCode                string `json:"ErrCode" xml:"ErrCode"`
	ErrMessage             string `json:"ErrMessage" xml:"ErrMessage"`
	RequestId              string `json:"RequestId" xml:"RequestId"`
	SubscriptionInstanceId string `json:"SubscriptionInstanceId" xml:"SubscriptionInstanceId"`
}

// CreateCreateSubscriptionInstanceRequest creates a request to invoke CreateSubscriptionInstance API
func CreateCreateSubscriptionInstanceRequest() (request *CreateSubscriptionInstanceRequest) {
	request = &CreateSubscriptionInstanceRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Dts", "2018-08-01", "CreateSubscriptionInstance", "dts", "openAPI")
	return
}

// CreateCreateSubscriptionInstanceResponse creates a response to parse from CreateSubscriptionInstance response
func CreateCreateSubscriptionInstanceResponse() (response *CreateSubscriptionInstanceResponse) {
	response = &CreateSubscriptionInstanceResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
