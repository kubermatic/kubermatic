package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"github.com/kubermatic/kubermatic/api/pkg/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getClusterUpgrades(updateManager UpdateManager, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		req, ok := request.(common.GetClusterReq)
		if !ok {
			return nil, errors.NewWrongRequest(request, common.GetClusterReq{})
		}

		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		versions, err := updateManager.GetPossibleUpdates(cluster.Spec.Version.String())
		if err != nil {
			return nil, err
		}
		var upgrades []*apiv1.MasterVersion
		for _, v := range versions {
			upgrades = append(upgrades, &apiv1.MasterVersion{
				Version: v.Version,
			})
		}

		return upgrades, nil
	}
}

func getClusterNodeUpgrades(updateManager UpdateManager, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		req, ok := request.(common.GetClusterReq)
		if !ok {
			return nil, errors.NewWrongRequest(request, common.GetClusterReq{})
		}

		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		versions, err := updateManager.GetMasterVersions()
		if err != nil {
			return nil, fmt.Errorf("failed to get master versions: %v", err)
		}

		var compatibleVersions []*version.MasterVersion
		clusterVersion := cluster.Spec.Version.Semver()
		for _, v := range versions {
			if err = ensureVersionCompatible(clusterVersion, v.Version); err == nil {
				compatibleVersions = append(compatibleVersions, v)
			} else {
				_, ok := err.(errVersionSkew)
				if !ok {
					return nil, fmt.Errorf("failed to check compatibility between kubelet %q and control plane %q: %v", v.Version, clusterVersion, err)
				}
			}
		}

		return convertVersionsToExternal(compatibleVersions), nil
	}
}

// UpgradeClusterNodeDeploymentsReq defines HTTP request for upgradeClusterNodeDeployments endpoint
// swagger:parameters upgradeClusterNodeDeployments
type UpgradeClusterNodeDeploymentsReq struct {
	common.GetClusterReq

	// in: body
	Body apiv1.MasterVersion
}

func DecodeUpgradeClusterNodeDeploymentsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req UpgradeClusterNodeDeploymentsReq
	cr, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(common.GetClusterReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func upgradeClusterNodeDeployments(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		req, ok := request.(UpgradeClusterNodeDeploymentsReq)
		if !ok {
			return nil, errors.NewWrongRequest(request, common.GetClusterReq{})
		}

		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machineClient, err := clusterProvider.GetAdminMachineClientForCustomerCluster(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machineDeployments, err := machineClient.ClusterV1alpha1().MachineDeployments(metav1.NamespaceSystem).List(metav1.ListOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		var updateErrors []error
		for _, machineDeployment := range machineDeployments.Items {
			machineDeployment.Spec.Template.Spec.Versions.Kubelet = req.Body.Version.String()
			_, err = machineClient.ClusterV1alpha1().MachineDeployments(metav1.NamespaceSystem).Update(&machineDeployment)
			if err != nil {
				updateErrors = append(updateErrors, err)
			}
		}

		if len(updateErrors) > 0 {
			return nil, fmt.Errorf("failed to update some node deployments: %+q", updateErrors)
		}

		return nil, nil
	}
}

func getMasterVersions(updateManager UpdateManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		versions, err := updateManager.GetMasterVersions()
		if err != nil {
			return nil, fmt.Errorf("failed to get master versions: %v", err)
		}
		return convertVersionsToExternal(versions), nil
	}
}

func convertVersionsToExternal(versions []*version.MasterVersion) []*apiv1.MasterVersion {
	sv := make([]*apiv1.MasterVersion, len(versions))
	for v := range versions {
		sv[v] = &apiv1.MasterVersion{
			Version: versions[v].Version,
			Default: versions[v].Default,
		}
	}
	return sv
}
