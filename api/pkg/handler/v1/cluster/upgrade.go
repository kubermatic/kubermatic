package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Masterminds/semver"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"github.com/kubermatic/kubermatic/api/pkg/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func GetUpgradesEndpoint(updateManager common.UpdateManager, projectProvider provider.ProjectProvider) endpoint.Endpoint {
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

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machineDeployments := &clusterv1alpha1.MachineDeploymentList{}
		if err := client.List(ctx, &ctrlruntimeclient.ListOptions{Namespace: metav1.NamespaceSystem}, machineDeployments); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		versions, err := updateManager.GetPossibleUpdates(cluster.Spec.Version.String())
		if err != nil {
			return nil, err
		}

		var upgrades []*apiv1.MasterVersion
		for _, v := range versions {
			isRestrictedByKubeletVersions, err := isRestrictedByKubeletVersions(v, machineDeployments.Items)
			if err != nil {
				return nil, err
			}

			upgrades = append(upgrades, &apiv1.MasterVersion{
				Version:                    v.Version,
				RestrictedByKubeletVersion: isRestrictedByKubeletVersions,
			})
		}

		return upgrades, nil
	}
}

func isRestrictedByKubeletVersions(controlPlaneVersion *version.MasterVersion, mds []clusterv1alpha1.MachineDeployment) (bool, error) {
	for _, md := range mds {
		kubeletVersion, err := semver.NewVersion(md.Spec.Template.Spec.Versions.Kubelet)
		if err != nil {
			return false, err
		}

		if err = common.EnsureVersionCompatible(controlPlaneVersion.Version, kubeletVersion); err != nil {
			return true, nil
		}
	}
	return false, nil
}

func GetNodeUpgradesEndpoint(updateManager common.UpdateManager, projectProvider provider.ProjectProvider) endpoint.Endpoint {
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

		compatibleVersions, err := filterIncompatibleVersions(versions, cluster.Spec.Version.Semver())
		if err != nil {
			return nil, fmt.Errorf("failed filter incompatible versions: %v", err)
		}

		return convertVersionsToExternal(compatibleVersions), nil
	}
}

// NodeUpgradesReq defines HTTP request for getNodeUpgrades
// swagger:parameters getNodeUpgrades
type NodeUpgradesReq struct {
	// in: body
	Body apiv1.MasterVersion
}

func decodeNodeUpgradesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NodeUpgradesReq
	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func getNodeUpgrades(updateManager common.UpdateManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(NodeUpgradesReq)
		if !ok {
			return nil, errors.NewWrongRequest(request, NodeUpgradesReq{})
		}

		versions, err := updateManager.GetMasterVersions()
		if err != nil {
			return nil, fmt.Errorf("failed to get master versions: %v", err)
		}

		compatibleVersions, err := filterIncompatibleVersions(versions, req.Body.Version)
		if err != nil {
			return nil, fmt.Errorf("failed filter incompatible versions: %v", err)
		}

		return convertVersionsToExternal(compatibleVersions), nil
	}
}

func filterIncompatibleVersions(possibleKubeletVersions []*version.MasterVersion, controlPlaneVersion *semver.Version) ([]*version.MasterVersion, error) {
	var compatibleVersions []*version.MasterVersion
	for _, v := range possibleKubeletVersions {
		if err := common.EnsureVersionCompatible(controlPlaneVersion, v.Version); err == nil {
			compatibleVersions = append(compatibleVersions, v)
		} else {
			_, ok := err.(common.ErrVersionSkew)
			if !ok {
				return nil, fmt.Errorf("failed to check compatibility between kubelet %q and control plane %q: %v", v.Version, controlPlaneVersion, err)
			}
		}
	}
	return compatibleVersions, nil
}

// UpgradeNodeDeploymentsReq defines HTTP request for upgradeClusterNodeDeployments endpoint
// swagger:parameters upgradeClusterNodeDeployments
type UpgradeNodeDeploymentsReq struct {
	common.GetClusterReq

	// in: body
	Body apiv1.MasterVersion
}

func DecodeUpgradeNodeDeploymentsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req UpgradeNodeDeploymentsReq
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

func UpgradeNodeDeploymentsEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		req, ok := request.(UpgradeNodeDeploymentsReq)
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

		requestedKubeletVersion, err := semver.NewVersion(req.Body.Version.String())
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		if err = common.EnsureVersionCompatible(cluster.Spec.Version.Version, requestedKubeletVersion); err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		client, err := clusterProvider.GetAdminClientForCustomerCluster(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machineDeployments := &clusterv1alpha1.MachineDeploymentList{}
		if err := client.List(ctx, &ctrlruntimeclient.ListOptions{Namespace: metav1.NamespaceSystem}, machineDeployments); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		var updateErrors []string
		for _, machineDeployment := range machineDeployments.Items {
			machineDeployment.Spec.Template.Spec.Versions.Kubelet = req.Body.Version.String()
			if err := client.Update(ctx, &machineDeployment); err != nil {
				updateErrors = append(updateErrors, err.Error())
			}
		}

		if len(updateErrors) > 0 {
			return nil, errors.NewWithDetails(http.StatusInternalServerError, "failed to update some node deployments", updateErrors)
		}

		return nil, nil
	}
}

func GetMasterVersionsEndpoint(updateManager common.UpdateManager) endpoint.Endpoint {
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
