/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provider

import (
	"fmt"
	"net/http"

	"github.com/packethost/packngo"
)

// Used to decode response object.
type plansRoot struct {
	Plans []packngo.Plan `json:"plans"`
}

func DescribePacketSize(apiKey, projectID, instanceType string) (packngo.Plan, error) {
	plan := packngo.Plan{}

	if len(apiKey) == 0 {
		return plan, fmt.Errorf("missing required parameter: apiKey")
	}

	if len(projectID) == 0 {
		return plan, fmt.Errorf("missing required parameter: projectID")
	}

	packetclient := packngo.NewClientWithAuth("kubermatic", apiKey, nil)
	req, err := packetclient.NewRequest(http.MethodGet, "/projects/"+projectID+"/plans", nil)
	if err != nil {
		return plan, err
	}
	root := new(plansRoot)

	_, err = packetclient.Do(req, root)
	if err != nil {
		return plan, err
	}

	plans := root.Plans
	for _, currentPlan := range plans {
		if currentPlan.Slug == instanceType {
			return currentPlan, nil
		}
	}
	return plan, fmt.Errorf("packet instanceType:%s not found", instanceType)
}
