package common

import (
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/semver"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/validation/nodeupdate"
	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CheckClusterVersionSkew returns a list of machines and/or machine deployments
// that are running kubelet at a version incompatible with the cluster's control plane.
func CheckClusterVersionSkew(ctx context.Context, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ClusterProvider, cluster *kubermaticapiv1.Cluster, projectID string) ([]string, error) {
	client, err := GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create a machine client: %v", err)
	}

	// get deduplicated list of all used kubelet versions
	kubeletVersions, err := getKubeletVersions(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to get the list of kubelet versions used in the cluster: %v", err)
	}

	// this is where the incompatible versions shall be saved
	incompatibleVersionsSet := map[string]bool{}

	clusterVersion := cluster.Spec.Version.Semver()
	for _, ver := range kubeletVersions {
		kubeletVersion, parseErr := semver.NewVersion(ver)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse kubelet version: %v", parseErr)
		}

		if err = nodeupdate.EnsureVersionCompatible(clusterVersion, kubeletVersion); err != nil {
			// errVersionSkew says it's incompatible
			if _, ok := err.(nodeupdate.ErrVersionSkew); ok {
				incompatibleVersionsSet[kubeletVersion.String()] = true
				continue
			}

			// other error types
			return nil, fmt.Errorf("failed to check compatibility between kubelet %q and control plane %q: %v", kubeletVersion, clusterVersion, err)
		}
	}

	// collect the deduplicated map entries into a slice
	var incompatibleVersionsList []string
	for ver := range incompatibleVersionsSet {
		incompatibleVersionsList = append(incompatibleVersionsList, ver)
	}

	return incompatibleVersionsList, nil
}

// getKubeletVersions returns the list of all kubelet versions used by a given cluster's Machines and MachineDeployments
func getKubeletVersions(ctx context.Context, client ctrlruntimeclient.Client) ([]string, error) {

	machineList := &clusterv1alpha1.MachineList{}
	if err := client.List(ctx, machineList); err != nil {
		return nil, fmt.Errorf("failed to load machines from cluster: %v", err)
	}

	machineDeployments := &clusterv1alpha1.MachineDeploymentList{}
	if err := client.List(ctx, machineDeployments); err != nil {
		return nil, KubernetesErrorToHTTPError(err)
	}

	kubeletVersionsSet := map[string]bool{}

	// first let's go through the legacy non-MD nodes
	for _, m := range machineList.Items {
		// Only list Machines that are not controlled, i.e. by Machine Set.
		if len(m.ObjectMeta.OwnerReferences) == 0 {
			ver := strings.TrimSpace(m.Spec.Versions.Kubelet)
			kubeletVersionsSet[ver] = true
		}
	}

	// now the deployments
	for _, md := range machineDeployments.Items {
		ver := strings.TrimSpace(md.Spec.Template.Spec.Versions.Kubelet)
		kubeletVersionsSet[ver] = true
	}

	// deduplicated list
	kubeletVersionList := []string{}
	for ver := range kubeletVersionsSet {
		kubeletVersionList = append(kubeletVersionList, ver)
	}

	return kubeletVersionList, nil
}
