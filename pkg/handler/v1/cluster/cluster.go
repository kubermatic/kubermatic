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

package cluster

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/errors"
	kubermaticerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	"k8s.io/klog"
)

func CreateEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, credentialManager provider.PresetProvider,
	exposeStrategy kubermaticv1.ExposeStrategy, userInfoGetter provider.UserInfoGetter, settingsProvider provider.SettingsProvider, updateManager common.UpdateManager, caBundle *x509.CertPool) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateReq)
		globalSettings, err := settingsProvider.GetGlobalSettings()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		err = req.Validate(globalSettings.Spec.ClusterTypeOptions, updateManager)
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		return handlercommon.CreateEndpoint(ctx, req.ProjectID, req.Body, projectProvider, privilegedProjectProvider, seedsGetter, credentialManager, exposeStrategy, userInfoGetter, caBundle)
	}
}

func GetEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		return handlercommon.GetEndpoint(ctx, projectProvider, privilegedProjectProvider, seedsGetter, userInfoGetter, req.ProjectID, req.ClusterID)
	}
}

func PatchEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter, caBundle *x509.CertPool) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(PatchReq)
		return handlercommon.PatchEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID, req.Patch, seedsGetter,
			projectProvider, privilegedProjectProvider, caBundle)
	}
}

// ListEndpoint list clusters within the given datacenter
func ListEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		apiClusters, err := handlercommon.GetExternalClusters(ctx, userInfoGetter, clusterProvider, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return apiClusters, nil
	}
}

// ListAllEndpoint list clusters for the given project in all datacenters
func ListAllEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, clusterProviderGetter provider.ClusterProviderGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetProjectRq)
		allClusters := make([]*apiv1.Cluster, 0)

		seeds, err := seedsGetter()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		for _, seed := range seeds {
			// if a Seed is bad, do not forward that error to the user, but only log
			clusterProvider, err := clusterProviderGetter(seed)
			if err != nil {
				klog.Errorf("failed to create cluster provider for seed %s: %v", seed.Name, err)
				continue
			}
			apiClusters, err := handlercommon.GetExternalClusters(ctx, userInfoGetter, clusterProvider, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			allClusters = append(allClusters, apiClusters...)
		}

		return allClusters, nil
	}
}

func DeleteEndpoint(sshKeyProvider provider.SSHKeyProvider, privilegedSSHKeyProvider provider.PrivilegedSSHKeyProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DeleteReq)
		return handlercommon.DeleteEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID, req.DeleteVolumes, req.DeleteLoadBalancers, sshKeyProvider, privilegedSSHKeyProvider, projectProvider, privilegedProjectProvider)
	}
}

func GetClusterEventsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(EventsReq)
		return handlercommon.GetClusterEventsEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID, req.Type, projectProvider, privilegedProjectProvider)
	}
}

func HealthEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		return handlercommon.HealthEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID, projectProvider, privilegedProjectProvider)
	}
}

func AssignSSHKeyEndpoint(sshKeyProvider provider.SSHKeyProvider, privilegedSSHKeyProvider provider.PrivilegedSSHKeyProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AssignSSHKeysReq)
		return handlercommon.AssignSSHKeyEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID, req.KeyID, projectProvider, privilegedProjectProvider, sshKeyProvider, privilegedSSHKeyProvider)
	}
}

func ListSSHKeysEndpoint(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListSSHKeysReq)
		return handlercommon.ListSSHKeysEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID, projectProvider, privilegedProjectProvider, sshKeyProvider)
	}
}

func DetachSSHKeyEndpoint(sshKeyProvider provider.SSHKeyProvider, privilegedSSHKeyProvider provider.PrivilegedSSHKeyProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DetachSSHKeysReq)
		return handlercommon.DetachSSHKeyEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID, req.KeyID, projectProvider, privilegedProjectProvider, sshKeyProvider, privilegedSSHKeyProvider)
	}
}

func GetMetricsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		return handlercommon.GetMetricsEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID, projectProvider, privilegedProjectProvider)
	}
}

// AssignSSHKeysReq defines HTTP request data for assignSSHKeyToCluster  endpoint
// swagger:parameters assignSSHKeyToCluster
type AssignSSHKeysReq struct {
	common.DCReq
	// in: path
	ClusterID string `json:"cluster_id"`
	// in: path
	KeyID string `json:"key_id"`
}

// ListSSHKeysReq defines HTTP request data for listSSHKeysAssignedToCluster endpoint
// swagger:parameters listSSHKeysAssignedToCluster
type ListSSHKeysReq struct {
	common.DCReq
	// in: path
	ClusterID string `json:"cluster_id"`
}

// DetachSSHKeysReq defines HTTP request for detachSSHKeyFromCluster endpoint
// swagger:parameters detachSSHKeyFromCluster
type DetachSSHKeysReq struct {
	common.DCReq
	// in: path
	KeyID string `json:"key_id"`
	// in: path
	ClusterID string `json:"cluster_id"`
}

// CreateReq defines HTTP request for createCluster endpoint
// swagger:parameters createCluster
type CreateReq struct {
	common.DCReq
	// in: body
	Body apiv1.CreateClusterSpec
}

// Validate validates DeleteEndpoint request
func (r CreateReq) Validate(clusterType kubermaticv1.ClusterType, updateManager common.UpdateManager) error {
	if len(r.ProjectID) == 0 || len(r.DC) == 0 {
		return fmt.Errorf("the service account ID and datacenter cannot be empty")
	}
	if r.Body.Cluster.ID != "" {
		return fmt.Errorf("cluster.ID is read-only")
	}

	if !handlercommon.ClusterTypes.Has(r.Body.Cluster.Type) {
		return fmt.Errorf("invalid cluster type %s", r.Body.Cluster.Type)
	}

	if clusterType != kubermaticv1.ClusterTypeAll && clusterType != apiv1.ToInternalClusterType(r.Body.Cluster.Type) {
		return fmt.Errorf("disabled cluster type %s", r.Body.Cluster.Type)
	}

	if r.Body.Cluster.Spec.Version.Version == nil {
		return fmt.Errorf("invalid cluster: invalid cloud spec \"Version\" is required but was not specified")
	}

	versions, err := updateManager.GetVersions(r.Body.Cluster.Type)
	if err != nil {
		return fmt.Errorf("failed to get available cluster versions: %v", err)
	}
	for _, availableVersion := range versions {
		if r.Body.Cluster.Spec.Version.Version.Equal(availableVersion.Version) {
			return nil
		}
	}

	return fmt.Errorf("invalid cluster: invalid cloud spec: unsupported version %v", r.Body.Cluster.Spec.Version.Version)
}

func DecodeCreateReq(c context.Context, r *http.Request) (interface{}, error) {
	var req CreateReq

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	if len(req.Body.Cluster.Type) == 0 {
		req.Body.Cluster.Type = apiv1.KubernetesClusterType
	}

	return req, nil
}

// ListReq defines HTTP request for listClusters endpoint
// swagger:parameters listClusters
type ListReq struct {
	common.DCReq
}

func DecodeListReq(c context.Context, r *http.Request) (interface{}, error) {
	var req ListReq

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)

	return req, nil
}

// PatchReq defines HTTP request for patchCluster endpoint
// swagger:parameters patchCluster
type PatchReq struct {
	common.GetClusterReq

	// in: body
	Patch json.RawMessage
}

func DecodePatchReq(c context.Context, r *http.Request) (interface{}, error) {
	var req PatchReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)
	req.ClusterID = clusterID

	if req.Patch, err = ioutil.ReadAll(r.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func DecodeAssignSSHKeyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AssignSSHKeysReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)

	keyID, err := common.DecodeSSHKeyID(c, r)
	if err != nil {
		return nil, err
	}
	req.KeyID = keyID

	return req, nil
}

func DecodeListSSHKeysReq(c context.Context, r *http.Request) (interface{}, error) {
	var req ListSSHKeysReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)

	return req, nil
}

func DecodeDetachSSHKeysReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DetachSSHKeysReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)

	sshKeyID, ok := mux.Vars(r)["key_id"]
	if !ok {
		return nil, fmt.Errorf("'key_id' parameter is required in order to delete ssh key")
	}
	req.KeyID = sshKeyID

	return req, nil
}

// AdminTokenReq defines HTTP request data for revokeClusterAdminToken and revokeClusterViewerToken endpoints.
// swagger:parameters revokeClusterAdminToken revokeClusterViewerToken
type AdminTokenReq struct {
	common.DCReq
	// in: path
	ClusterID string `json:"cluster_id"`
}

func DecodeAdminTokenReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AdminTokenReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)

	return req, nil
}

func RevokeAdminTokenEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AdminTokenReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		return nil, common.KubernetesErrorToHTTPError(clusterProvider.RevokeAdminKubeconfig(cluster))
	}
}

func RevokeViewerTokenEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AdminTokenReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		return nil, common.KubernetesErrorToHTTPError(clusterProvider.RevokeViewerKubeconfig(cluster))
	}
}

// DeleteReq defines HTTP request for deleteCluster endpoints
// swagger:parameters deleteCluster
type DeleteReq struct {
	common.GetClusterReq
	// in: header
	// DeleteVolumes if true all cluster PV's and PVC's will be deleted from cluster
	DeleteVolumes bool
	// in: header
	// DeleteLoadBalancers if true all load balancers will be deleted from cluster
	DeleteLoadBalancers bool
}

func DecodeDeleteReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DeleteReq

	clusterReqRaw, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	clusterReq := clusterReqRaw.(common.GetClusterReq)
	req.GetClusterReq = clusterReq

	headerValue := r.Header.Get("DeleteVolumes")
	if len(headerValue) > 0 {
		deleteVolumes, err := strconv.ParseBool(headerValue)
		if err != nil {
			return nil, err
		}
		req.DeleteVolumes = deleteVolumes
	}

	headerValue = r.Header.Get("DeleteLoadBalancers")
	if len(headerValue) > 0 {
		deleteLB, err := strconv.ParseBool(headerValue)
		if err != nil {
			return nil, err
		}
		req.DeleteLoadBalancers = deleteLB
	}

	return req, nil
}

// EventsReq defines HTTP request for getClusterEvents endpoint
// swagger:parameters getClusterEvents
type EventsReq struct {
	common.GetClusterReq

	// in: query
	Type string `json:"type,omitempty"`
}

func DecodeGetClusterEvents(c context.Context, r *http.Request) (interface{}, error) {
	var req EventsReq

	clusterReqRaw, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	clusterReq := clusterReqRaw.(common.GetClusterReq)
	req.GetClusterReq = clusterReq

	req.Type = r.URL.Query().Get("type")
	if len(req.Type) > 0 {
		if req.Type == "warning" || req.Type == "normal" {
			return req, nil
		}
		return nil, fmt.Errorf("wrong query parameter, unsupported type: %s", req.Type)
	}

	return req, nil
}

func ListNamespaceEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		return handlercommon.ListNamespaceEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID, projectProvider, privilegedProjectProvider)
	}
}

// GetClusterProviderFromRequest returns cluster and cluster provider based on the provided request.
func GetClusterProviderFromRequest(
	ctx context.Context,
	request interface{},
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	userInfoGetter provider.UserInfoGetter) (*kubermaticv1.Cluster, *kubernetesprovider.ClusterProvider, error) {

	req, ok := request.(common.GetClusterReq)
	if !ok {
		return nil, nil, kubermaticerrors.New(http.StatusBadRequest, "invalid request")
	}

	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
	if err != nil {
		return nil, nil, kubermaticerrors.New(http.StatusInternalServerError, err.Error())
	}

	rawClusterProvider, ok := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)
	if !ok {
		return nil, nil, kubermaticerrors.New(http.StatusInternalServerError, "no clusterProvider in request")
	}
	clusterProvider, ok := rawClusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, nil, kubermaticerrors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}
	return cluster, clusterProvider, nil
}
