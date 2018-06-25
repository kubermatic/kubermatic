package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/golang/glog"
	prometheusapi "github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"github.com/kubermatic/kubermatic/api/pkg/validation"

	"github.com/gorilla/mux"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

// Deprecated: newClusterEndpoint is deprecated use newCreateClusterEndpoint instead.
func newClusterEndpoint(sshKeysProvider provider.SSHKeyProvider, cloudProviders map[string]provider.CloudProvider, updateManager UpdateManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NewClusterReq)
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

func newCreateClusterEndpoint(sshKeyProvider provider.NewSSHKeyProvider, cloudProviders map[string]provider.CloudProvider, updateManager UpdateManager, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		// decode the request
		req := request.(NewClusterReq)
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		clusterProvider := ctx.Value(newClusterProviderContextKey).(provider.NewClusterProvider)
		project, err := projectProvider.Get(user, req.ProjectName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		// validate the request
		// note that: the ssh keys are optional when provided make sure that they belong to the given project.
		if len(req.Body.SSHKeys) > 0 {
			projectSSHKeys, err := sshKeyProvider.List(user, project, nil)
			if err != nil {
				return nil, kubernetesErrorToHTTPError(err)
			}

			for _, requestSSHKeyName := range req.Body.SSHKeys {
				found := false
				for _, projectSSHKey := range projectSSHKeys {
					if projectSSHKey.Name == requestSSHKeyName {
						found = true
						break
					}
				}
				if !found {
					return nil, fmt.Errorf("the given ssh key %s does not belong to the given project %s (%s)", requestSSHKeyName, project.Spec.Name, project.Name)
				}
			}
		}
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

		// create the cluster and assign the requested ssh keys to the newly created cluster
		newCluster, err := clusterProvider.New(project, user, spec)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		for _, requestSSHKeyName := range req.Body.SSHKeys {
			sshKey, err := sshKeyProvider.Get(user, project, requestSSHKeyName)
			if err != nil {
				return nil, kubernetesErrorToHTTPError(err)
			}
			if sshKey.IsUsedByCluster(newCluster.Name) {
				continue
			}
			sshKey.AddToCluster(newCluster.Name)
			if _, err := sshKeyProvider.Update(user, project, sshKey); err != nil {
				return nil, kubernetesErrorToHTTPError(err)
			}
		}

		return nil, nil
	}
}

// Deprecated: clusterEndpoint is deprecated use newGetCluster instead.
func clusterEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := ctx.Value(apiUserContextKey).(apiv1.User)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)
		req := request.(NewGetClusterReq)
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
		return cluster, nil
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
		req := request.(UpdateClusterReq)
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

		//We don't allow updating the following fields
		newCluster := req.Body.Cluster
		newCluster.TypeMeta = existingCluster.TypeMeta
		newCluster.ObjectMeta = existingCluster.ObjectMeta
		newCluster.Status = existingCluster.Status

		if err := validation.ValidateUpdateCluster(newCluster, existingCluster, cloudProviders); err != nil {
			return nil, errors.NewBadRequest("invalid cluster: %v", err)
		}

		return clusterProvider.Update(user, project, newCluster)
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

		clusters, err := clusterProvider.List(project)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		return clusters, nil
	}
}

// Deprecated: deleteClusterEndpoint is deprecated use newDeleteCluster instead.
func deleteClusterEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NewGetClusterReq)
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

func newDeleteCluster(projectProvider provider.ProjectProvider) endpoint.Endpoint {
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
		req := request.(NewGetClusterReq)
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

func newDecodeCreateClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NewClusterReq

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

func newDecodeUpdateClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req UpdateClusterReq
	clusterName, ok := mux.Vars(r)["cluster_name"]
	if !ok {
		return "", fmt.Errorf("'cluster_name' parameter is required but was not provided")
	}
	req.ClusterName = clusterName

	dcr, err := decodeDcReq(c, r)
	req.DCReq = dcr.(DCReq)

	if err != nil {
		return nil, err
	}
	projectName, err := decodeProjectPathReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectName = projectName

	if err := json.NewDecoder(r.Body).Decode(&req.Body.Cluster); err != nil {
		return nil, err
	}

	return req, nil
}
