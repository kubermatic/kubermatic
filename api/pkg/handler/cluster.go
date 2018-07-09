package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	prometheusapi "github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"github.com/kubermatic/kubermatic/api/pkg/validation"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

// Deprecated: newClusterEndpoint is deprecated use newCreateClusterEndpoint instead.
func newClusterEndpoint(sshKeysProvider provider.SSHKeyProvider, cloudProviders map[string]provider.CloudProvider, updateManager UpdateManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ClusterReq)
		user := ctx.Value(apiUserContextKey).(apiv1.User)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)

		spec := req.Body.Cluster
		if spec == nil {
			return nil, errors.NewBadRequest("no cluster spec given")
		}

		if err := validation.ValidateCreateClusterSpec(spec, cloudProviders); err != nil {
			return nil, errors.NewBadRequest("invalid cluster: %v", err)
		}

		if spec.Version == "" {
			v, err := updateManager.GetDefault()
			if err != nil {
				return nil, err
			}
			spec.Version = v.Version.String()
		}

		c, err := clusterProvider.NewCluster(user, spec)
		if err != nil {
			if kerrors.IsAlreadyExists(err) {
				return nil, errors.NewConflict("cluster", spec.Cloud.DatacenterName, spec.HumanReadableName)
			}
			return nil, err
		}

		err = sshKeysProvider.AssignSSHKeysToCluster(user, req.Body.SSHKeys, c.Name)
		if err != nil {
			return nil, err
		}

		return c, nil
	}
}

func newCreateClusterEndpoint(sshKeyProvider provider.NewSSHKeyProvider, cloudProviders map[string]provider.CloudProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NewCreateClusterReq)
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		clusterProvider := ctx.Value(newClusterProviderContextKey).(provider.NewClusterProvider)
		project, err := projectProvider.Get(user, req.ProjectName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		spec := &kubermaticapiv1.ClusterSpec{}
		spec.HumanReadableName = req.Body.Name
		spec.Cloud = &req.Body.Spec.Cloud
		spec.Version = req.Body.Spec.Version
		if err := validation.ValidateCreateClusterSpec(spec, cloudProviders); err != nil {
			return nil, errors.NewBadRequest("invalid cluster: %v", err)
		}

		existingClusters, err := clusterProvider.List(project, &provider.ClusterListOptions{ClusterName: spec.HumanReadableName})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		if len(existingClusters) > 0 {
			return nil, errors.NewAlreadyExists("cluster", spec.HumanReadableName)
		}

		newCluster, err := clusterProvider.New(project, user, spec)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		return newCluster, nil
	}
}

// Deprecated: clusterEndpoint is deprecated use newGetCluster instead.
func clusterEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := ctx.Value(apiUserContextKey).(apiv1.User)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)
		req := request.(GetClusterReq)
		c, err := clusterProvider.Cluster(user, req.ClusterName)
		if err != nil {
			if err == provider.ErrNotFound {
				return nil, errors.NewNotFound("cluster", req.ClusterName)
			}
			return nil, err
		}

		return c, nil
	}
}

func newGetCluster(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NewGetClusterReq)
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		clusterProvider := ctx.Value(newClusterProviderContextKey).(provider.NewClusterProvider)
		project, err := projectProvider.Get(user, req.ProjectName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		cluster, err := clusterProvider.Get(user, project, req.ClusterName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		apiClusters := convertInternalClustersToExternal([]*kubermaticapiv1.Cluster{cluster})
		if len(apiClusters) != 1 {
			return nil, errors.New(http.StatusInternalServerError, "unable to convert cluster resource")

		}
		return apiClusters[0], nil
	}
}

// Deprecated: updateClusterEndpoint is deprecated use newUpdateCluster instead.
func updateClusterEndpoint(cloudProviders map[string]provider.CloudProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := ctx.Value(apiUserContextKey).(apiv1.User)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)
		req := request.(UpdateClusterReq)
		oldCluster, err := clusterProvider.Cluster(user, req.ClusterName)
		if err != nil {
			if err == provider.ErrNotFound {
				return nil, errors.NewNotFound("cluster", req.ClusterName)
			}
			return nil, err
		}
		newCluster := req.Body.Cluster

		//We don't allow updating the following fields
		newCluster.TypeMeta = oldCluster.TypeMeta
		newCluster.ObjectMeta = oldCluster.ObjectMeta
		newCluster.Status = oldCluster.Status

		if err := validation.ValidateUpdateCluster(newCluster, oldCluster, cloudProviders); err != nil {
			return nil, errors.NewBadRequest("invalid cluster: %v", err)
		}

		return clusterProvider.UpdateCluster(user, newCluster)
	}
}

func newUpdateCluster(cloudProviders map[string]provider.CloudProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NewUpdateClusterReq)
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		clusterProvider := ctx.Value(newClusterProviderContextKey).(provider.NewClusterProvider)
		project, err := projectProvider.Get(user, req.ProjectName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		existingCluster, err := clusterProvider.Get(user, project, req.ClusterName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		existingCluster.Spec.Cloud = &req.Body.Spec.Cloud
		existingCluster.Spec.Version = req.Body.Spec.Version

		if err := validation.ValidateUpdateCluster(existingCluster, existingCluster, cloudProviders); err != nil {
			return nil, errors.NewBadRequest("invalid cluster: %v", err)
		}

		updatedCluster, err := clusterProvider.Update(user, project, existingCluster)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		convertedClusters := convertInternalClustersToExternal([]*kubermaticapiv1.Cluster{updatedCluster})
		if len(convertedClusters) != 1 {
			return nil, errors.New(http.StatusInternalServerError, "unable to convert cluster resource")
		}
		return convertedClusters[0], nil
	}
}

// Deprecated: clustersEndpoint is deprecated use newListClusters instead.
func clustersEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := ctx.Value(apiUserContextKey).(apiv1.User)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)
		cs, err := clusterProvider.Clusters(user)
		if err != nil {
			return nil, err
		}

		return cs, nil
	}
}

func newListClusters(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NewListClustersReq)
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		clusterProvider := ctx.Value(newClusterProviderContextKey).(provider.NewClusterProvider)
		project, err := projectProvider.Get(user, req.ProjectName)
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

// Deprecated: deleteClusterEndpoint is deprecated use newDeleteCluster instead.
func deleteClusterEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetClusterReq)
		user := ctx.Value(apiUserContextKey).(apiv1.User)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)
		c, err := clusterProvider.Cluster(user, req.ClusterName)
		if err != nil {
			if err == provider.ErrNotFound {
				return nil, errors.NewNotFound("cluster", req.ClusterName)
			}
			return nil, err
		}

		return nil, clusterProvider.DeleteCluster(user, c.Name)
	}
}

func newDeleteCluster(sshKeyProvider provider.NewSSHKeyProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NewGetClusterReq)
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		clusterProvider := ctx.Value(newClusterProviderContextKey).(provider.NewClusterProvider)
		project, err := projectProvider.Get(user, req.ProjectName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		err = clusterProvider.Delete(user, project, req.ClusterName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		// TODO: I think that in general it would be better if the cluster resource
		// has the reference to the ssh keys - not the other way around as it is now.
		// detach ssh keys that are being used by this clusters
		clusterSSHKeys, err := sshKeyProvider.List(user, project, &provider.SSHKeyListOptions{ClusterName: req.ClusterName})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		for _, clusterSSHKey := range clusterSSHKeys {
			clusterSSHKey.RemoveFromCluster(req.ClusterName)
			if _, err := sshKeyProvider.Update(user, project, clusterSSHKey); err != nil {
				return nil, kubernetesErrorToHTTPError(err)
			}
		}
		return nil, nil
	}
}

func getClusterHealth(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NewGetClusterReq)
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		clusterProvider := ctx.Value(newClusterProviderContextKey).(provider.NewClusterProvider)
		project, err := projectProvider.Get(user, req.ProjectName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		existingCluster, err := clusterProvider.Get(user, project, req.ClusterName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		return apiv1.NewClusterHealth{
			Apiserver:         existingCluster.Status.Health.Apiserver,
			Scheduler:         existingCluster.Status.Health.Scheduler,
			Controller:        existingCluster.Status.Health.Controller,
			MachineController: existingCluster.Status.Health.MachineController,
			Etcd:              existingCluster.Status.Health.Etcd,
		}, nil
	}
}

func assignSSHKeyToCluster(sshKeyProvider provider.NewSSHKeyProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AssignSSHKeysToClusterReq)
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		clusterProvider := ctx.Value(newClusterProviderContextKey).(provider.NewClusterProvider)
		if len(req.KeyName) == 0 {
			return nil, errors.NewBadRequest("please provide an SSH key")
		}

		project, err := projectProvider.Get(user, req.ProjectName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		_, err = clusterProvider.Get(user, project, req.ClusterName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		// sanity check, make sure that the key belongs to the project
		// alternatively we could examine the owner references
		{
			projectSSHKeys, err := sshKeyProvider.List(user, project, nil)
			if err != nil {
				return nil, kubernetesErrorToHTTPError(err)
			}

			found := false
			for _, projectSSHKey := range projectSSHKeys {
				if projectSSHKey.Name == req.KeyName {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("the given ssh key %s does not belong to the given project %s (%s)", req.KeyName, project.Spec.Name, project.Name)
			}
		}

		sshKey, err := sshKeyProvider.Get(user, project, req.KeyName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		if sshKey.IsUsedByCluster(req.ClusterName) {
			return nil, nil
		}
		sshKey.AddToCluster(req.ClusterName)
		_, err = sshKeyProvider.Update(user, project, sshKey)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		return nil, nil
	}
}

func listSSHKeysAssingedToCluster(sshKeyProvider provider.NewSSHKeyProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListSSHKeysAssignedToClusterReq)
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		clusterProvider := ctx.Value(newClusterProviderContextKey).(provider.NewClusterProvider)

		project, err := projectProvider.Get(user, req.ProjectName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		_, err = clusterProvider.Get(user, project, req.ClusterName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		keys, err := sshKeyProvider.List(user, project, &provider.SSHKeyListOptions{ClusterName: req.ClusterName, SortBy: "metadata.creationTimestamp"})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		apiKeys := convertInternalSSHKeysToExternal(keys)
		return apiKeys, nil
	}
}

func convertInternalClustersToExternal(internalClusters []*kubermaticapiv1.Cluster) []*apiv1.NewCluster {
	apiClusters := make([]*apiv1.NewCluster, len(internalClusters))
	for index, cluster := range internalClusters {
		apiClusters[index] = &apiv1.NewCluster{
			NewObjectMeta: apiv1.NewObjectMeta{
				ID:                cluster.Name,
				Name:              cluster.Spec.HumanReadableName,
				CreationTimestamp: cluster.CreationTimestamp.Time,
				DeletionTimestamp: func() *time.Time {
					if cluster.DeletionTimestamp != nil {
						return &cluster.DeletionTimestamp.Time
					}
					return nil
				}(),
			},
			Spec: apiv1.NewClusterSpec{
				Cloud:   *cluster.Spec.Cloud,
				Version: cluster.Spec.Version,
			},
			Status: apiv1.NewClusterStatus{
				Version: cluster.Spec.Version,
				URL:     cluster.Address.URL,
			},
		}
	}
	return apiClusters
}

func detachSSHKeyFromCluster(sshKeyProvider provider.NewSSHKeyProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DetachSSHKeysFromClusterReq)
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		clusterProvider := ctx.Value(newClusterProviderContextKey).(provider.NewClusterProvider)
		project, err := projectProvider.Get(user, req.ProjectName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		_, err = clusterProvider.Get(user, project, req.ClusterName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		// sanity check, make sure that the key belongs to the project
		// alternatively we could examine the owner references
		{
			projectSSHKeys, err := sshKeyProvider.List(user, project, nil)
			if err != nil {
				return nil, kubernetesErrorToHTTPError(err)
			}

			found := false
			for _, projectSSHKey := range projectSSHKeys {
				if projectSSHKey.Name == req.KeyName {
					found = true
					break
				}
			}
			if !found {
				return nil, errors.NewNotFound("sshkey", req.KeyName)
			}
		}

		clusterSSHKey, err := sshKeyProvider.Get(user, project, req.KeyName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		clusterSSHKey.RemoveFromCluster(req.ClusterName)
		_, err = sshKeyProvider.Update(user, project, clusterSSHKey)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		return nil, nil
	}
}

type (
	metricsResponse struct {
		Metrics []metricResponse `json:"metrics"`
	}
	metricResponse struct {
		Name   string    `json:"name"`
		Value  float64   `json:"value,omitempty"`
		Values []float64 `json:"values,omitempty"`
	}
)

func getClusterMetricsEndpoint(prometheusURL *string) endpoint.Endpoint {
	if prometheusURL == nil {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			return nil, fmt.Errorf("metrics endpoint disabled")
		}
	}

	promClient, err := prometheusapi.NewClient(prometheusapi.Config{
		Address: *prometheusURL,
	})
	if err != nil {
		glog.Fatal(err)
	}
	promAPI := prometheusv1.NewAPI(promClient)

	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := ctx.Value(apiUserContextKey).(apiv1.User)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)
		req := request.(GetClusterReq)
		c, err := clusterProvider.Cluster(user, req.ClusterName)
		if err != nil {
			if err == provider.ErrNotFound {
				return nil, errors.NewNotFound("cluster", req.ClusterName)
			}
			return nil, err
		}

		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		var resp metricsResponse

		val, err := prometheusQuery(ctx, promAPI, fmt.Sprintf(`sum(machine_controller_machines{cluster="%s"})`, c.Name))
		if err != nil {
			return nil, err
		}
		resp.Metrics = append(resp.Metrics, metricResponse{
			Name:  "Machines",
			Value: val,
		})

		vals, err := prometheusQueryRange(ctx, promAPI, fmt.Sprintf(`sum(machine_controller_machines{cluster="%s"})`, c.Name))
		if err != nil {
			return nil, err
		}
		resp.Metrics = append(resp.Metrics, metricResponse{
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
	assignSSHKeysToClusterBodyReq
	// in: path
	ProjectName string `json:"project_id"`
	// in: path
	ClusterName string `json:"cluster_name"`
}

type assignSSHKeysToClusterBodyReq struct {
	KeyName string `json:"KeyName"`
}

// ListSSHKeysAssignedToClusterReq defines HTTP request data for listSSHKeysAssignedToCluster endpoint
// swagger:parameters listSSHKeysAssignedToCluster
type ListSSHKeysAssignedToClusterReq struct {
	DCReq
	// in: path
	ProjectName string `json:"project_id"`
	// in: path
	ClusterName string `json:"cluster_name"`
}

// DetachSSHKeysFromClusterReq defines HTTP request for detachSSHKeyFromCluster endpoint
// swagger:parameters detachSSHKeyFromCluster
type DetachSSHKeysFromClusterReq struct {
	DCReq
	// in: path
	KeyName string `json:"key_name"`
	// in: path
	ProjectName string `json:"project_id"`
	// in: path
	ClusterName string `json:"cluster_name"`
}

// NewCreateClusterReq defines HTTP request for newCreateCluster endpoint
// swagger:parameters newCreateCluster
type NewCreateClusterReq struct {
	DCReq
	// in: body
	Body apiv1.NewCluster
	// in: path
	ProjectName string `json:"project_id"`
}

func newDecodeCreateClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NewCreateClusterReq

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	projectName, err := decodeProjectPathReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectName = projectName

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// NewListClustersReq defines HTTP request for newListClsters endpoint
// swagger:parameters newListClusters
type NewListClustersReq struct {
	DCReq
	// in: path
	ProjectName string `json:"project_id"`
}

func newDecodeListClustersReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NewListClustersReq

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	projectName, err := decodeProjectPathReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectName = projectName

	return req, nil
}

// NewGetClusterReq defines HTTP request for newDeleteCluster and newGetClusterKubeconfig endpoints
// swagger:parameters newGetCluster newDeleteCluster newGetClusterKubeconfig newGetClusterHealth
type NewGetClusterReq struct {
	DCReq
	// in: path
	ClusterName string `json:"cluster_name"`
	// in: path
	ProjectName string `json:"project_id"`
}

func newDecodeGetClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NewGetClusterReq
	clusterName, ok := mux.Vars(r)["cluster_name"]
	if !ok {
		return "", fmt.Errorf("'cluster_name' parameter is required but was not provided")
	}
	req.ClusterName = clusterName

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	projectName, err := decodeProjectPathReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectName = projectName

	return req, nil
}

func decodeClusterNameAndProject(c context.Context, r *http.Request) (string, string, error) {
	clusterName, ok := mux.Vars(r)["cluster_name"]
	if !ok {
		return "", "", fmt.Errorf("'cluster_name' parameter is required but was not provided")
	}

	projectName, err := decodeProjectPathReq(c, r)
	if err != nil {
		return "", "", err
	}
	return clusterName, projectName, nil
}

// NewUpdateClusterReq defines HTTP request for newUpdateCluster endpoint
// swagger:parameters newUpdateCluster
type NewUpdateClusterReq struct {
	NewGetClusterReq
	// in: body
	Body apiv1.NewCluster
}

func newDecodeUpdateClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NewUpdateClusterReq
	clusterName, projectName, err := decodeClusterNameAndProject(c, r)
	if err != nil {
		return nil, err
	}

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)
	req.ClusterName = clusterName
	req.ProjectName = projectName

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func decodeAssignSSHKeyToClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AssignSSHKeysToClusterReq
	clusterName, projectName, err := decodeClusterNameAndProject(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterName = clusterName
	req.ProjectName = projectName

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	if err := json.NewDecoder(r.Body).Decode(&req.assignSSHKeysToClusterBodyReq); err != nil {
		return nil, err
	}

	return req, nil
}

func decodeListSSHKeysAssignedToCluster(c context.Context, r *http.Request) (interface{}, error) {
	var req ListSSHKeysAssignedToClusterReq
	clusterName, projectName, err := decodeClusterNameAndProject(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterName = clusterName
	req.ProjectName = projectName

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	return req, nil
}

func decodeDetachSSHKeysFromCluster(c context.Context, r *http.Request) (interface{}, error) {
	var req DetachSSHKeysFromClusterReq
	clusterName, projectName, err := decodeClusterNameAndProject(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterName = clusterName
	req.ProjectName = projectName

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	sshKeyName, ok := mux.Vars(r)["key_name"]
	if !ok {
		return nil, fmt.Errorf("'key_name' parameter is required in order to delete ssh key")
	}
	req.KeyName = sshKeyName

	return req, nil
}
