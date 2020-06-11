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
