package handler

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1alpha1clientset "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
)

// errVersionSkew denotes an error condition where a given kubelet/controlplane version pair is not supported
type errVersionSkew struct {
	ControlPlane *semver.Version
	Kubelet      *semver.Version
}

// Error returns a string representation of the error
func (e errVersionSkew) Error() string {
	return fmt.Sprintf("kubelet version %s is not compatible with control plane version %s", e.Kubelet, e.ControlPlane)
}

var _ error = errVersionSkew{}

// ensureVersionCompatible checks whether the given kubelet version
// is deemed compatible with the given version of the control plane.
func ensureVersionCompatible(controlPlane *semver.Version, kubelet *semver.Version) error {
	if controlPlane == nil {
		return errors.New("ensureVersionCompatible: controlPlane is nil")
	}

	if kubelet == nil {
		return errors.New("ensureVersionCompatible: kubelet is nil")
	}

	// Kubelet must be the same major version and no more than 2 minor versions behind the control plane.
	// https://kubernetes.io/docs/setup/version-skew-policy/
	// https://github.com/kubernetes/website/blob/076efdf364651859553681a75f60c957de729023/content/en/docs/setup/version-skew-policy.md
	compatible := kubelet.Major() == controlPlane.Major() && kubelet.Minor() >= (controlPlane.Minor()-2) && kubelet.Minor() <= controlPlane.Minor()

	if !compatible {
		return errVersionSkew{
			ControlPlane: controlPlane,
			Kubelet:      kubelet,
		}
	}

	return nil
}

// checkClusterVersionSkew returns a list of machines and/or machine deployments
// that are running kubelet at a version incompatible with the cluster's control plane.
func checkClusterVersionSkew(_ *provider.UserInfo, clusterProvider provider.ClusterProvider, cluster *kubermaticapiv1.Cluster) ([]string, error) {
	machineClient, err := clusterProvider.GetAdminMachineClientForCustomerCluster(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to create a machine client: %v", err)
	}

	// get deduplicated list of all used kubelet versions
	kubeletVersions, err := getKubeletVersions(machineClient)
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

		if err = ensureVersionCompatible(clusterVersion, kubeletVersion); err != nil {
			// errVersionSkew says it's incompatible
			if _, ok := err.(errVersionSkew); ok {
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
func getKubeletVersions(machineClient clusterv1alpha1clientset.Interface) ([]string, error) {
	machineList, err := machineClient.ClusterV1alpha1().Machines(metav1.NamespaceSystem).List(metav1.ListOptions{IncludeUninitialized: true})
	if err != nil {
		return nil, fmt.Errorf("failed to load machines from cluster: %v", err)
	}

	machineDeployments, err := machineClient.ClusterV1alpha1().MachineDeployments(metav1.NamespaceSystem).List(metav1.ListOptions{})
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
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
