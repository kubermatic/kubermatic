/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package nutanix

import (
	"fmt"
	"strings"

	nutanixv3 "github.com/embik/nutanix-client-go/pkg/client/v3"
)

func GetClusters(client *ClientSet) ([]nutanixv3.ClusterIntentResponse, error) {
	resp, err := client.Prism.V3.ListAllCluster("")
	if err != nil {
		return nil, wrapNutanixError(err)
	}

	var clusters []nutanixv3.ClusterIntentResponse

	if resp != nil {
		for _, entity := range resp.Entities {
			if entity != nil {
				clusters = append(clusters, *entity)
			}
		}
	}

	return clusters, nil
}

func GetProjects(client *ClientSet) ([]nutanixv3.Project, error) {
	resp, err := client.Prism.V3.ListAllProject("")
	if err != nil {
		return nil, wrapNutanixError(err)
	}

	var projects []nutanixv3.Project

	if resp != nil {
		for _, entity := range resp.Entities {
			if entity != nil {
				projects = append(projects, *entity)
			}
		}
	}

	return projects, nil
}

func GetSubnets(client *ClientSet, clusterName, projectName string) ([]nutanixv3.SubnetIntentResponse, error) {
	resp, err := client.Prism.V3.ListAllSubnet("")
	if err != nil {
		return nil, wrapNutanixError(err)
	}

	var subnets []nutanixv3.SubnetIntentResponse

	if resp != nil {
		for _, entity := range resp.Entities {
			if entity != nil {
				if entity.Status != nil && entity.Status.ClusterReference != nil && *entity.Status.ClusterReference.Name == clusterName &&
					(projectName == "" || (entity.Metadata != nil && entity.Metadata.ProjectReference != nil && *entity.Metadata.ProjectReference.Name == projectName)) {
					subnets = append(subnets, *entity)
				}
			}
		}
	}

	return subnets, nil
}

func wrapNutanixError(initialErr error) error {
	if initialErr == nil {
		return nil
	}

	resp, err := ParseNutanixError(initialErr)
	if err != nil {
		// failed to parse error, let's make sure it doesnt't have new lines at least
		return fmt.Errorf("api error: %s", strings.ReplaceAll(initialErr.Error(), "\n", ""))
	}

	var msgs []string
	for _, msg := range resp.MessageList {
		msgs = append(msgs, fmt.Sprintf("%s: %s", msg.Message, msg.Reason))
	}

	return fmt.Errorf("api error (%s, code %d): %s", resp.State, resp.Code, strings.Join(msgs, ", "))
}
