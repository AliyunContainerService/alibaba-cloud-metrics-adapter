package ess

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

// ScalingActivity is a nested struct in ess response
type ScalingActivity struct {
	ScalingActivityId     string `json:"ScalingActivityId" xml:"ScalingActivityId"`
	ScalingGroupId        string `json:"ScalingGroupId" xml:"ScalingGroupId"`
	Description           string `json:"Description" xml:"Description"`
	Cause                 string `json:"Cause" xml:"Cause"`
	StartTime             string `json:"StartTime" xml:"StartTime"`
	EndTime               string `json:"EndTime" xml:"EndTime"`
	Progress              int    `json:"Progress" xml:"Progress"`
	StatusCode            string `json:"StatusCode" xml:"StatusCode"`
	StatusMessage         string `json:"StatusMessage" xml:"StatusMessage"`
	TotalCapacity         string `json:"TotalCapacity" xml:"TotalCapacity"`
	AttachedCapacity      string `json:"AttachedCapacity" xml:"AttachedCapacity"`
	AutoCreatedCapacity   string `json:"AutoCreatedCapacity" xml:"AutoCreatedCapacity"`
	ScalingInstanceNumber int    `json:"ScalingInstanceNumber" xml:"ScalingInstanceNumber"`
}
