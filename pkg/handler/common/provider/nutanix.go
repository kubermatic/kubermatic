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

package provider

import (
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	nutanixprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/nutanix"
)

type NutanixCredentials struct {
	Endpoint      string
	Port          *int32
	AllowInsecure *bool
	ProxyURL      string
	Username      string
	Password      string
}

func ListNutanixClusters(creds NutanixCredentials) (apiv1.NutanixClusterList, error) {
	clientSet, err := nutanixprovider.GetClientSetWithCreds(creds.Endpoint, creds.Port, creds.AllowInsecure, creds.ProxyURL, creds.Username, creds.Password)
	if err != nil {
		return nil, err
	}

	clusterResp, err := nutanixprovider.GetClusters(clientSet)
	if err != nil {
		return nil, err
	}

	var clusters apiv1.NutanixClusterList
	for _, cluster := range clusterResp {
		clusters = append(clusters, apiv1.NutanixCluster{
			Name: *cluster.Status.Name,
		})
	}

	return clusters, nil
}

func ListNutanixProjects(creds NutanixCredentials) (apiv1.NutanixProjectList, error) {
	clientSet, err := nutanixprovider.GetClientSetWithCreds(creds.Endpoint, creds.Port, creds.AllowInsecure, creds.ProxyURL, creds.Username, creds.Password)
	if err != nil {
		return nil, err
	}

	projectsResp, err := nutanixprovider.GetProjects(clientSet)
	if err != nil {
		return nil, err
	}

	var projects apiv1.NutanixProjectList
	for _, cluster := range projectsResp {
		projects = append(projects, apiv1.NutanixProject{
			Name: cluster.Status.Name,
		})
	}

	return projects, nil
}

func ListNutanixSubnets(creds NutanixCredentials, clusterName, projectName string) (apiv1.NutanixSubnetList, error) {
	clientSet, err := nutanixprovider.GetClientSetWithCreds(creds.Endpoint, creds.Port, creds.AllowInsecure, creds.ProxyURL, creds.Username, creds.Password)
	if err != nil {
		return nil, err
	}

	subnetResp, err := nutanixprovider.GetSubnets(clientSet, clusterName, projectName)
	if err != nil {
		return nil, err
	}

	var subnets apiv1.NutanixSubnetList
	for _, subnet := range subnetResp {
		subnets = append(subnets, apiv1.NutanixSubnet{
			Name:   *subnet.Status.Name,
			Type:   *subnet.Status.Resources.SubnetType,
			VlanID: int(*subnet.Status.Resources.VlanID),
		})
	}

	return subnets, nil
}
