package vod

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

// Project is a nested struct in vod response
type Project struct {
	ModifiedTime    string  `json:"ModifiedTime" xml:"ModifiedTime"`
	RegionId        string  `json:"RegionId" xml:"RegionId"`
	Title           string  `json:"Title" xml:"Title"`
	Duration        float64 `json:"Duration" xml:"Duration"`
	ProjectId       string  `json:"ProjectId" xml:"ProjectId"`
	StorageLocation string  `json:"StorageLocation" xml:"StorageLocation"`
	CreationTime    string  `json:"CreationTime" xml:"CreationTime"`
	Status          string  `json:"Status" xml:"Status"`
	Description     string  `json:"Description" xml:"Description"`
	Timeline        string  `json:"Timeline" xml:"Timeline"`
	CoverURL        string  `json:"CoverURL" xml:"CoverURL"`
}
