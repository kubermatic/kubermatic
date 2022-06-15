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

package kubernetesdashboard

import (
	"net/http"

	"github.com/gorilla/mux"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

type InitialRequest struct {
	// Embed the original request
	*http.Request

	// in: query
	ProjectID string `json:"projectID"`
	ClusterID string `json:"clusterID"`
}

func (this *InitialRequest) decode(r *http.Request) *InitialRequest {
	this.ProjectID = r.URL.Query().Get("projectID")
	this.ClusterID = r.URL.Query().Get("clusterID")

	this.Request = r

	return this
}

func NewInitialRequest(r *http.Request) *InitialRequest {
	result := new(InitialRequest)
	return result.decode(r)
}

type OIDCCallbackRequest struct {
	// Embed the original request
	*http.Request

	// in: query
	Code  string `json:"code"`
	State string `json:"state"`
}

func (this *OIDCCallbackRequest) decode(r *http.Request) *OIDCCallbackRequest {
	this.Code = r.URL.Query().Get("code")
	this.State = r.URL.Query().Get("state")
	this.Request = r

	return this
}

func NewOIDCCallbackRequest(r *http.Request) *OIDCCallbackRequest {
	result := new(OIDCCallbackRequest)
	return result.decode(r)
}

type ProxyRequest struct {
	// Embed the original request
	*http.Request

	// in: path
	ProjectID string `json:"project_id"`
	ClusterID string `json:"cluster_id"`

	// in: query
	Token string `json:"token"`
}

func (this *ProxyRequest) decode(r *http.Request) *ProxyRequest {
	// Path params
	this.ProjectID = mux.Vars(r)["project_id"]
	this.ClusterID = mux.Vars(r)["cluster_id"]

	// Query params
	this.Token = r.URL.Query().Get("token")

	// Embed original request
	this.Request = r

	return this
}

// GetSeedCluster implements the middleware.seedClusterGetter interface.
func (this *ProxyRequest) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: this.ClusterID,
	}
}

func NewProxyRequest(r *http.Request) *ProxyRequest {
	result := new(ProxyRequest)
	return result.decode(r)
}
