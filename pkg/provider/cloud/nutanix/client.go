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
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	nutanixclient "github.com/embik/nutanix-client-go/pkg/client"
	nutanixv3 "github.com/embik/nutanix-client-go/pkg/client/v3"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
)

type ClientSet struct {
	Prism *nutanixv3.Client
}

type ErrorResponse struct {
	APIVersion  string             `json:"api_version"`
	Kind        string             `json:"kind"`
	State       string             `json:"state"`
	MessageList []ErrorResponseMsg `json:"message_list"`
	Code        int32              `json:"code"`
}

type ErrorResponseMsg struct {
	Message string `json:"message"`
	Reason  string `json:"reason"`
}

func GetClientSet(dc *kubermaticv1.DatacenterSpecNutanix, cloud *kubermaticv1.NutanixCloudSpec, secretKeyGetter provider.SecretKeySelectorValueFunc) (*ClientSet, error) {
	return getClientSet(dc, cloud, secretKeyGetter)
}

func getCredentials(dc *kubermaticv1.DatacenterSpecNutanix, cloud *kubermaticv1.NutanixCloudSpec, secretKeyGetter provider.SecretKeySelectorValueFunc) (nutanixclient.Credentials, error) {
	username := cloud.Username
	password := cloud.Password

	var err error

	if username == "" {
		if cloud.CredentialsReference == nil {
			return nutanixclient.Credentials{}, errors.New("no credentials provided")
		}
		username, err = secretKeyGetter(cloud.CredentialsReference, resources.NutanixUsername)
		if err != nil {
			return nutanixclient.Credentials{}, err
		}
	}

	if password == "" {
		if cloud.CredentialsReference == nil {
			return nutanixclient.Credentials{}, errors.New("no credentials provided")
		}
		password, err = secretKeyGetter(cloud.CredentialsReference, resources.NutanixPassword)
		if err != nil {
			return nutanixclient.Credentials{}, err
		}
	}

	port := 9440
	if dc.Port != nil {
		port = int(*dc.Port)
	}

	creds := nutanixclient.Credentials{
		URL:      net.JoinHostPort(dc.Endpoint, fmt.Sprint(port)),
		Endpoint: dc.Endpoint,
		Port:     strconv.Itoa(port),
		Username: username,
		Password: password,
		Insecure: dc.AllowInsecure,
	}

	// set up proxy URL if it's given through the cloud spec

	proxyURL := cloud.ProxyURL
	if proxyURL == "" && cloud.CredentialsReference != nil {
		credsProxyURL, _ := secretKeyGetter(cloud.CredentialsReference, resources.NutanixProxyURL)
		if credsProxyURL != "" {
			proxyURL = credsProxyURL
		}
	}

	if proxyURL != "" {
		creds.ProxyURL = proxyURL
	}

	return creds, nil
}

func getClientSet(dc *kubermaticv1.DatacenterSpecNutanix, cloud *kubermaticv1.NutanixCloudSpec, secretKeyGetter provider.SecretKeySelectorValueFunc) (*ClientSet, error) {
	credentials, err := getCredentials(dc, cloud, secretKeyGetter)
	if err != nil {
		return nil, err
	}

	clientV3, err := nutanixv3.NewV3Client(credentials)
	if err != nil {
		return nil, err
	}

	return &ClientSet{
		Prism: clientV3,
	}, nil
}

func GetSubnetByName(client *ClientSet, name, clusterID string) (*nutanixv3.SubnetIntentResponse, error) {
	filter := fmt.Sprintf("name==%s", name)
	subnets, err := client.Prism.V3.ListAllSubnet(filter)

	if err != nil {
		return nil, err
	}

	if subnets == nil || subnets.Entities == nil {
		return nil, fmt.Errorf("subnets list is nil for '%s'", filter)
	}

	for _, subnet := range subnets.Entities {
		if subnet == nil {
			return nil, errors.New("subnet is nil")
		}

		if subnet.Status == nil {
			return nil, errors.New("subnet status is nil")
		}

		if subnet.Status.Name == nil {
			return nil, errors.New("subnet name is nil")
		}

		if subnet.Status.ClusterReference == nil || subnet.Status.ClusterReference.UUID == nil {
			return nil, errors.New("subnet status does not contain valid cluster reference")
		}

		if *subnet.Status.Name == name && *subnet.Status.ClusterReference.UUID == clusterID {
			return subnet, nil
		}
	}

	return nil, fmt.Errorf("no subnet found for '%s' on cluster '%s'", filter, clusterID)
}

func GetProjectByName(client *ClientSet, name string) (*nutanixv3.Project, error) {
	filter := fmt.Sprintf("name==%s", name)
	projects, err := client.Prism.V3.ListAllProject(filter)

	if err != nil {
		return nil, err
	}

	if projects == nil || projects.Entities == nil {
		return nil, fmt.Errorf("projects list is nil for '%s'", filter)
	}

	for _, project := range projects.Entities {
		if project == nil {
			return nil, errors.New("project is nil")
		}

		if project.Status == nil {
			return nil, errors.New("project status is nil")
		}

		if project.Status.Name == name {
			return project, nil
		}
	}

	return nil, fmt.Errorf("no project found for '%s'", filter)
}

func GetClusterByName(client *ClientSet, name string) (*nutanixv3.ClusterIntentResponse, error) {
	filter := fmt.Sprintf("name==%s", name)
	clusters, err := client.Prism.V3.ListAllCluster(filter)

	if err != nil {
		return nil, err
	}

	if clusters == nil || clusters.Entities == nil {
		return nil, fmt.Errorf("clusters list is nil for '%s'", filter)
	}

	for _, cluster := range clusters.Entities {
		if cluster == nil {
			return nil, errors.New("cluster is nil")
		}

		if cluster.Status == nil {
			return nil, errors.New("cluster status is nil")
		}

		if cluster.Status.Name == nil {
			return nil, errors.New("cluster name is nil")
		}

		if *cluster.Status.Name == name {
			return cluster, nil
		}
	}

	return nil, fmt.Errorf("no cluster found for '%s'", filter)
}

func ParseNutanixError(err error) (*ErrorResponse, error) {
	if err == nil {
		return nil, nil
	}

	// the api returns a json error ... but with a string prefixed to it
	errJsonString := strings.TrimPrefix(err.Error(), "error: ")

	var resp ErrorResponse

	if parseErr := json.Unmarshal([]byte(errJsonString), &resp); parseErr != nil {
		return nil, fmt.Errorf("failed to parse '%v': %w", err, parseErr)
	}

	return &resp, nil
}
