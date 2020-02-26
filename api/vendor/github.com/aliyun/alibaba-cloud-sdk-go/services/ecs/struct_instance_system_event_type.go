package ecs

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

// InstanceSystemEventType is a nested struct in ecs response
type InstanceSystemEventType struct {
	InstanceId        string            `json:"InstanceId" xml:"InstanceId"`
	EventId           string            `json:"EventId" xml:"EventId"`
	EventPublishTime  string            `json:"EventPublishTime" xml:"EventPublishTime"`
	NotBefore         string            `json:"NotBefore" xml:"NotBefore"`
	EventFinishTime   string            `json:"EventFinishTime" xml:"EventFinishTime"`
	EventType         EventType         `json:"EventType" xml:"EventType"`
	EventCycleStatus  EventCycleStatus  `json:"EventCycleStatus" xml:"EventCycleStatus"`
	ExtendedAttribute ExtendedAttribute `json:"ExtendedAttribute" xml:"ExtendedAttribute"`
}
