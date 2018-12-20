package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/middleware"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/evanphx/json-patch"
	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	prometheusapi "github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources/cluster"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"github.com/kubermatic/kubermatic/api/pkg/validation"
)

func createClusterEndpoint(cloudProviders map[string]provider.CloudProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateClusterReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		spec, err := cluster.Spec(req.Body, cloudProviders)
		if err != nil {
			return nil, errors.NewBadRequest("invalid cluster: %v", err)
		}

		existingClusters, err := clusterProvider.List(project, &provider.ClusterListOptions{ClusterSpecName: spec.HumanReadableName})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		if len(existingClusters) > 0 {
			return nil, errors.NewAlreadyExists("cluster", spec.HumanReadableName)
		}

		newCluster, err := clusterProvider.New(project, userInfo, spec)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		return convertInternalClusterToExternal(newCluster), nil
	}
}

func getCluster(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetClusterReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		return convertInternalClusterToExternal(cluster), nil
	}
}

func patchCluster(cloudProviders map[string]provider.CloudProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(PatchClusterReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		existingCluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		existingClusterJSON, err := json.Marshal(existingCluster)
		if err != nil {
			return nil, errors.NewBadRequest("cannot decode existing cluster: %v", err)
		}

		patchedClusterJSON, err := jsonpatch.MergePatch(existingClusterJSON, req.Patch)
		if err != nil {
			return nil, errors.NewBadRequest("cannot patch cluster: %v", err)
		}

		var patchedCluster *kubermaticapiv1.Cluster
		err = json.Unmarshal(patchedClusterJSON, &patchedCluster)
		if err != nil {
			return nil, errors.NewBadRequest("cannot decode patched cluster: %v", err)
		}

		if err = validation.ValidateUpdateCluster(patchedCluster, existingCluster, cloudProviders); err != nil {
			return nil, errors.NewBadRequest("invalid cluster: %v", err)
		}

		updatedCluster, err := clusterProvider.Update(userInfo, patchedCluster)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		return convertInternalClusterToExternal(updatedCluster), nil
	}
}

func listClusters(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListClustersReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		clusters, err := clusterProvider.List(project, nil)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		apiClusters := convertInternalClustersToExternal(clusters)
		return apiClusters, nil
	}
}

func deleteCluster(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetClusterReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		clusterSSHKeys, err := sshKeyProvider.List(project, &provider.SSHKeyListOptions{ClusterName: req.ClusterID})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		for _, clusterSSHKey := range clusterSSHKeys {
			clusterSSHKey.RemoveFromCluster(req.ClusterID)
			if _, err = sshKeyProvider.Update(userInfo, clusterSSHKey); err != nil {
				return nil, kubernetesErrorToHTTPError(err)
			}
		}

		err = clusterProvider.Delete(userInfo, req.ClusterID)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		return nil, nil
	}
}

func getClusterHealth(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetClusterReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		existingCluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		return apiv1.ClusterHealth{
			Apiserver:         existingCluster.Status.Health.Apiserver,
			Scheduler:         existingCluster.Status.Health.Scheduler,
			Controller:        existingCluster.Status.Health.Controller,
			MachineController: existingCluster.Status.Health.MachineController,
			Etcd:              existingCluster.Status.Health.Etcd,
		}, nil
	}
}

func assignSSHKeyToCluster(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AssignSSHKeysToClusterReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		if len(req.KeyID) == 0 {
			return nil, errors.NewBadRequest("please provide an SSH key")
		}

		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		_, err = clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: false})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		// sanity check, make sure that the key belongs to the project
		// alternatively we could examine the owner references
		{
			projectSSHKeys, err := sshKeyProvider.List(project, nil)
			if err != nil {
				return nil, kubernetesErrorToHTTPError(err)
			}
			found := false
			for _, projectSSHKey := range projectSSHKeys {
				if projectSSHKey.Name == req.KeyID {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("the given ssh key %s does not belong to the given project %s (%s)", req.KeyID, project.Spec.Name, project.Name)
			}
		}

		sshKey, err := sshKeyProvider.Get(userInfo, req.KeyID)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		if sshKey.IsUsedByCluster(req.ClusterID) {
			return nil, nil
		}
		sshKey.AddToCluster(req.ClusterID)
		_, err = sshKeyProvider.Update(userInfo, sshKey)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		return nil, nil
	}
}

func listSSHKeysAssingedToCluster(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListSSHKeysAssignedToClusterReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		_, err = clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		keys, err := sshKeyProvider.List(project, &provider.SSHKeyListOptions{ClusterName: req.ClusterID})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		apiKeys := convertInternalSSHKeysToExternal(keys)
		return apiKeys, nil
	}
}

func convertInternalClusterToExternal(internalCluster *kubermaticapiv1.Cluster) *apiv1.Cluster {
	return &apiv1.Cluster{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                internalCluster.Name,
			Name:              internalCluster.Spec.HumanReadableName,
			CreationTimestamp: internalCluster.CreationTimestamp.Time,
			DeletionTimestamp: func() *time.Time {
				if internalCluster.DeletionTimestamp != nil {
					return &internalCluster.DeletionTimestamp.Time
				}
				return nil
			}(),
		},
		Spec: apiv1.ClusterSpec{
			Cloud:           internalCluster.Spec.Cloud,
			Version:         internalCluster.Spec.Version,
			MachineNetworks: internalCluster.Spec.MachineNetworks,
		},
		Status: apiv1.ClusterStatus{
			Version: internalCluster.Spec.Version,
			URL:     internalCluster.Address.URL,
		},
	}
}

func convertInternalClustersToExternal(internalClusters []*kubermaticapiv1.Cluster) []*apiv1.Cluster {
	apiClusters := make([]*apiv1.Cluster, len(internalClusters))
	for index, cluster := range internalClusters {
		apiClusters[index] = convertInternalClusterToExternal(cluster)
	}
	return apiClusters
}

func detachSSHKeyFromCluster(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DetachSSHKeysFromClusterReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		_, err = clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		// sanity check, make sure that the key belongs to the project
		// alternatively we could examine the owner references
		{
			projectSSHKeys, err := sshKeyProvider.List(project, nil)
			if err != nil {
				return nil, kubernetesErrorToHTTPError(err)
			}

			found := false
			for _, projectSSHKey := range projectSSHKeys {
				if projectSSHKey.Name == req.KeyID {
					found = true
					break
				}
			}
			if !found {
				return nil, errors.NewNotFound("sshkey", req.KeyID)
			}
		}

		clusterSSHKey, err := sshKeyProvider.Get(userInfo, req.KeyID)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		clusterSSHKey.RemoveFromCluster(req.ClusterID)
		_, err = sshKeyProvider.Update(userInfo, clusterSSHKey)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		return nil, nil
	}
}

func getClusterMetrics(projectProvider provider.ProjectProvider, prometheusClient prometheusapi.Client) endpoint.Endpoint {
	if prometheusClient == nil {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			return nil, fmt.Errorf("metrics endpoint disabled")
		}
	}

	promAPI := prometheusv1.NewAPI(prometheusClient)

	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetClusterReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		c, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		var resp []apiv1.ClusterMetric

		val, err := prometheusQuery(ctx, promAPI, fmt.Sprintf(`sum(machine_controller_machines{cluster="%s"})`, c.Name))
		if err != nil {
			return nil, err
		}
		resp = append(resp, apiv1.ClusterMetric{
			Name:   "Machines",
			Values: []float64{val},
		})

		vals, err := prometheusQueryRange(ctx, promAPI, fmt.Sprintf(`sum(machine_controller_machines{cluster="%s"})`, c.Name))
		if err != nil {
			return nil, err
		}
		resp = append(resp, apiv1.ClusterMetric{
			Name:   "Machines (1h)",
			Values: vals,
		})

		return resp, nil
	}
}

func prometheusQuery(ctx context.Context, api prometheusv1.API, query string) (float64, error) {
	now := time.Now()
	val, err := api.Query(ctx, query, now)
	if err != nil {
		return 0, nil
	}
	if val.Type() != model.ValVector {
		return 0, fmt.Errorf("failed to retrieve correct value type")
	}

	vec := val.(model.Vector)
	for _, sample := range vec {
		return float64(sample.Value), nil
	}

	return 0, nil
}

func prometheusQueryRange(ctx context.Context, api prometheusv1.API, query string) ([]float64, error) {
	now := time.Now()
	val, err := api.QueryRange(ctx, query, prometheusv1.Range{
		Start: now.Add(-1 * time.Hour),
		End:   now,
		Step:  30 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	if val.Type() != model.ValMatrix {
		return nil, fmt.Errorf("failed to retrieve correct value type")
	}

	var vals []float64
	matrix := val.(model.Matrix)
	for _, sample := range matrix {
		for _, v := range sample.Values {
			vals = append(vals, float64(v.Value))
		}
	}

	return vals, nil
}

// AssignSSHKeysToClusterReq defines HTTP request data for assignSSHKeyToCluster  endpoint
// swagger:parameters assignSSHKeyToCluster
type AssignSSHKeysToClusterReq struct {
	DCReq
	// in: path
	ClusterID string `json:"cluster_id"`
	// in: path
	KeyID string `json:"key_id"`
}

// ListSSHKeysAssignedToClusterReq defines HTTP request data for listSSHKeysAssignedToCluster endpoint
// swagger:parameters listSSHKeysAssignedToCluster
type ListSSHKeysAssignedToClusterReq struct {
	DCReq
	// in: path
	ClusterID string `json:"cluster_id"`
}

// DetachSSHKeysFromClusterReq defines HTTP request for detachSSHKeyFromCluster endpoint
// swagger:parameters detachSSHKeyFromCluster
type DetachSSHKeysFromClusterReq struct {
	DCReq
	// in: path
	KeyID string `json:"key_id"`
	// in: path
	ClusterID string `json:"cluster_id"`
}

// CreateClusterReq defines HTTP request for createCluster endpoint
// swagger:parameters createCluster
type CreateClusterReq struct {
	DCReq
	// in: body
	Body apiv1.Cluster
}

func decodeCreateClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req CreateClusterReq

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// ListClustersReq defines HTTP request for listClusters endpoint
// swagger:parameters listClusters
type ListClustersReq struct {
	DCReq
}

func decodeListClustersReq(c context.Context, r *http.Request) (interface{}, error) {
	var req ListClustersReq

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	return req, nil
}

// GetClusterReq defines HTTP request for deleteCluster and getClusterKubeconfig endpoints
// swagger:parameters getCluster deleteCluster getClusterKubeconfig getClusterHealth getNodeForCluster getClusterUpgrades getClusterMetrics
type GetClusterReq struct {
	DCReq
	// in: path
	ClusterID string `json:"cluster_id"`
}

func decodeGetClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GetClusterReq
	clusterID, err := decodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	return req, nil
}

func decodeClusterID(c context.Context, r *http.Request) (string, error) {
	clusterID := mux.Vars(r)["cluster_id"]
	if clusterID == "" {
		return "", fmt.Errorf("'cluster_id' parameter is required but was not provided")
	}

	return clusterID, nil
}

func decodeSSHKeyID(c context.Context, r *http.Request) (string, error) {
	keyID := mux.Vars(r)["key_id"]
	if keyID == "" {
		return "", fmt.Errorf("'key_id' parameter is required but was not provided")
	}

	return keyID, nil
}

// PatchClusterReq defines HTTP request for patchCluster endpoint
// swagger:parameters patchCluster
type PatchClusterReq struct {
	GetClusterReq

	// in: body
	Patch []byte
}

func decodePatchClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req PatchClusterReq
	clusterID, err := decodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)
	req.ClusterID = clusterID

	if req.Patch, err = ioutil.ReadAll(r.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func decodeAssignSSHKeyToClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AssignSSHKeysToClusterReq
	clusterID, err := decodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	keyID, err := decodeSSHKeyID(c, r)
	if err != nil {
		return nil, err
	}
	req.KeyID = keyID

	return req, nil
}

func decodeListSSHKeysAssignedToCluster(c context.Context, r *http.Request) (interface{}, error) {
	var req ListSSHKeysAssignedToClusterReq
	clusterID, err := decodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	return req, nil
}

func decodeDetachSSHKeysFromCluster(c context.Context, r *http.Request) (interface{}, error) {
	var req DetachSSHKeysFromClusterReq
	clusterID, err := decodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	sshKeyID, ok := mux.Vars(r)["key_id"]
	if !ok {
		return nil, fmt.Errorf("'key_id' parameter is required in order to delete ssh key")
	}
	req.KeyID = sshKeyID

	return req, nil
}

// ClusterAdminTokenReq defines HTTP request data for revokeClusterAdminToken endpoints.
// swagger:parameters revokeClusterAdminToken
type ClusterAdminTokenReq struct {
	DCReq
	// in: path
	ClusterID string `json:"cluster_id"`
}

func decodeClusterAdminTokenReq(c context.Context, r *http.Request) (interface{}, error) {
	var req ClusterAdminTokenReq
	clusterID, err := decodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	return req, nil
}

func revokeClusterAdminToken(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ClusterAdminTokenReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		cluster.Address.AdminToken = kubernetes.GenerateToken()

		_, err = clusterProvider.Update(userInfo, cluster)
		return nil, kubernetesErrorToHTTPError(err)
	}
}
