package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Masterminds/semver"
	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"github.com/kubermatic/kubermatic/api/pkg/validation/nodeupdate"
	"github.com/kubermatic/kubermatic/api/pkg/version"
	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetUpgradesEndpoint(updateManager common.UpdateManager, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		req, ok := request.(common.GetClusterReq)
		if !ok {
			return nil, errors.NewWrongRequest(request, common.GetClusterReq{})
		}
		cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machineDeployments := &clusterv1alpha1.MachineDeploymentList{}
		if err := client.List(ctx, machineDeployments, ctrlruntimeclient.InNamespace(metav1.NamespaceSystem)); err != nil {
			// Happens during cluster creation when the CRD is not setup yet
			if _, ok := err.(*meta.NoKindMatchError); ok {
				return nil, nil
			}
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		clusterType := apiv1.KubernetesClusterType
		if cluster.IsOpenshift() {
			clusterType = apiv1.OpenShiftClusterType
		}

		versions, err := updateManager.GetPossibleUpdates(cluster.Spec.Version.String(), clusterType)
		if err != nil {
			return nil, err
		}

		var upgrades []*apiv1.MasterVersion
		for _, v := range versions {
			isRestricted := false
			if clusterType == apiv1.KubernetesClusterType {
				isRestricted, err = isRestrictedByKubeletVersions(v, machineDeployments.Items)
				if err != nil {
					return nil, err
				}
			}

			upgrades = append(upgrades, &apiv1.MasterVersion{
				Version:                    v.Version,
				RestrictedByKubeletVersion: isRestricted,
			})
		}

		return upgrades, nil
	}
}

func isRestrictedByKubeletVersions(controlPlaneVersion *version.Version, mds []clusterv1alpha1.MachineDeployment) (bool, error) {
	for _, md := range mds {
		kubeletVersion, err := semver.NewVersion(md.Spec.Template.Spec.Versions.Kubelet)
		if err != nil {
			return false, err
		}

		if err = nodeupdate.EnsureVersionCompatible(controlPlaneVersion.Version, kubeletVersion); err != nil {
			return true, nil
		}
	}
	return false, nil
}

// NodeUpgradesReq defines HTTP request for getNodeUpgrades
// swagger:parameters getNodeUpgrades
type NodeUpgradesReq struct {
	TypeReq
	// in: query
	ControlPlaneVersion string `json:"control_plane_version,omitempty"`
}

func DecodeNodeUpgradesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NodeUpgradesReq

	clusterTypeReq, err := DecodeClusterTypeReq(c, r)
	if err != nil {
		return nil, err
	}
	req.TypeReq = clusterTypeReq.(TypeReq)

	req.ControlPlaneVersion = r.URL.Query().Get("control_plane_version")

	return req, nil
}

func GetNodeUpgrades(updateManager common.UpdateManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(NodeUpgradesReq)
		if !ok {
			return nil, errors.NewWrongRequest(request, NodeUpgradesReq{})
		}
		err := req.TypeReq.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		controlPlaneVersion, err := semver.NewVersion(req.ControlPlaneVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to parse control plane version: %v", err)
		}

		versions, err := updateManager.GetVersions(req.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to get master versions: %v", err)
		}

		compatibleVersions, err := filterIncompatibleVersions(versions, controlPlaneVersion)
		if err != nil {
			return nil, fmt.Errorf("failed filter incompatible versions: %v", err)
		}

		return convertVersionsToExternal(compatibleVersions), nil
	}
}

func filterIncompatibleVersions(possibleKubeletVersions []*version.Version, controlPlaneVersion *semver.Version) ([]*version.Version, error) {
	var compatibleVersions []*version.Version
	for _, v := range possibleKubeletVersions {
		if err := nodeupdate.EnsureVersionCompatible(controlPlaneVersion, v.Version); err == nil {
			compatibleVersions = append(compatibleVersions, v)
		} else {
			_, ok := err.(nodeupdate.ErrVersionSkew)
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

func UpgradeNodeDeploymentsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		req, ok := request.(UpgradeNodeDeploymentsReq)
		if !ok {
			return nil, errors.NewWrongRequest(request, common.GetClusterReq{})
		}
		cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		requestedKubeletVersion, err := semver.NewVersion(req.Body.Version.String())
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		if err = nodeupdate.EnsureVersionCompatible(cluster.Spec.Version.Version, requestedKubeletVersion); err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		client, err := clusterProvider.GetAdminClientForCustomerCluster(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machineDeployments := &clusterv1alpha1.MachineDeploymentList{}
		if err := client.List(ctx, machineDeployments, ctrlruntimeclient.InNamespace(metav1.NamespaceSystem)); err != nil {
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
		req := request.(TypeReq)
		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		versions, err := updateManager.GetVersions(req.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to get master versions: %v", err)
		}
		return convertVersionsToExternal(versions), nil
	}
}

// TypeReq represents a request that contains the cluster type
type TypeReq struct {
	// in: query
	Type string `json:"type"`
}

func (r TypeReq) Validate() error {
	if clusterTypes.Has(r.Type) {
		return nil
	}
	return fmt.Errorf("invalid cluster type %s", r.Type)
}

// DecodeAddReq  decodes an HTTP request into TypeReq
func DecodeClusterTypeReq(c context.Context, r *http.Request) (interface{}, error) {
	var req TypeReq

	req.Type = r.URL.Query().Get("type")
	if len(req.Type) == 0 {
		req.Type = apiv1.KubernetesClusterType
	}

	return req, nil
}

func convertVersionsToExternal(versions []*version.Version) []*apiv1.MasterVersion {
	sv := make([]*apiv1.MasterVersion, len(versions))
	for v := range versions {
		sv[v] = &apiv1.MasterVersion{
			Version: versions[v].Version,
			Default: versions[v].Default,
		}
	}
	return sv
}
