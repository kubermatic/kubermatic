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

package client

import (
	"encoding/json"
	"fmt"
)

func (c *Client) ListClusters(filters Filters) (*ClusterList, error) {
	endpoint, err := appendFilters(fmt.Sprintf("%s/v3/clusters/", c.options.Endpoint), filters)
	if err != nil {
		return nil, err
	}
	list := &ClusterList{}
	err = c.do(endpoint, "", list)
	return list, err
}

func (c *Client) CreateClusterRegistrationToken(token *ClusterRegistrationToken) (*ClusterRegistrationToken, error) {
	endpoint := fmt.Sprintf("%s/v3/clusterregistrationtokens", c.options.Endpoint)
	data, err := json.Marshal(token)
	if err != nil {
		return nil, fmt.Errorf("failed marshal clusterRegistrationToken object: %v", err)
	}
	err = c.do(endpoint, string(data), token)
	return token, err
}

func (c *Client) CreateImportedCluster(cluster *Cluster) (*Cluster, error) {
	endpoint := fmt.Sprintf("%s/v3/clusters/", c.options.Endpoint)
	data, err := json.Marshal(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed marshal cluster object: %v", err)
	}
	err = c.do(endpoint, string(data), cluster)
	return cluster, err
}
