package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/kit/endpoint"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"github.com/kubermatic/kubermatic/api/pkg/validation"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

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

func clusterEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := ctx.Value(apiUserContextKey).(apiv1.User)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)
		req := request.(ClusterReq)
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

func deleteClusterEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ClusterReq)
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

func getClusterMetricsEndpoint(promAPI prometheusv1.API) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := ctx.Value(apiUserContextKey).(apiv1.User)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)
		req := request.(ClusterReq)
		c, err := clusterProvider.Cluster(user, req.ClusterName)
		if err != nil {
			if err == provider.ErrNotFound {
				return nil, errors.NewNotFound("cluster", req.ClusterName)
			}
			return nil, err
		}

		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		vals, err := promMachines(ctx, promAPI, c.Name)

		return vals, nil
	}
}

func promMachines(ctx context.Context, api prometheusv1.API, clusterName string) ([]float64, error) {
	query := fmt.Sprintf(`machine_controller_machines{namespace="cluster-%s"}`, clusterName)

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

	}

	return nil, nil
}
