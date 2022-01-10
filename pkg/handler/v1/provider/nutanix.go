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
	"context"
	"net/http"
	"strconv"
	//"github.com/go-kit/kit/endpoint"
	//"k8c.io/kubermatic/v2/pkg/provider"
)

type NutanixCommonReq struct {
	// in: header
	// name: NutanixEndpoint
	Endpoint string

	// in: header
	// name: NutanixPort
	Port int

	// in: header
	// name: AllowInsecure
	AllowInsecure bool

	// in: header
	// name: NutanixUsername
	Username string

	// in: header
	// name: NutanixPassword
	Password string

	// in: header
	// name: ProxyURL
	ProxyURL string

	// in: header
	// name: Credential
	Credential string
}

// NutanixClusterReq represent a request for Nutanix clusters
// swagger:parameters listNutanixClusters
type NutanixClusterReq struct {
	NutanixCommonReq
}

// NutanixSubnetReq represent a request for Nutanix subnets
// swagger:parameters listNutanixSubnets
type NutanixSubnetReq struct {
	NutanixCommonReq

	// in: path
	// required: true
	Cluster string `json:"cluster"`

	// Project query parameter. Can be omitted to query subnets without project scope
	// in: query
	Project string `json:"project,omitempty"`
}

func DecodeNutanixCommonReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NutanixCommonReq

	req.Endpoint = r.Header.Get("NutanixEndpoint")
	req.Username = r.Header.Get("NutanixUsername")
	req.Password = r.Header.Get("NutanixPassword")
	req.ProxyURL = r.Header.Get("ProxyURL")
	req.Credential = r.Header.Get("Credential")

	port, err := strconv.Atoi(r.Header.Get("NutanixPort"))
	if err != nil {
		return nil, err
	}
	req.Port = port

	allowInsecure, err := strconv.ParseBool(r.Header.Get("AllowInsecure"))
	if err != nil {
		return nil, err
	}
	req.AllowInsecure = allowInsecure

	return req, nil
}

func DecodeNutanixSubnetReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NutanixSubnetReq

	commonReq, err := DecodeNutanixCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.NutanixCommonReq = commonReq.(NutanixCommonReq)

	return req, nil
}

/*
func NutanixClusterEndpoint(presetProvider provider.PresetProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {

}
*/
