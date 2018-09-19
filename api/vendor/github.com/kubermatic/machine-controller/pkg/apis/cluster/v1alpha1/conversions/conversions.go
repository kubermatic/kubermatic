package conversions

import (
	"encoding/json"
	"fmt"

	machinesv1alpha1 "github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"

	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

const (
	ContainerRuntimeInfoKey = "containerRuntimeInfo"

	TypeRevisionAnnotationName = "machine-controller/machine-type-revision"

	TypeRevisionCurrentVersion = "e1903be683739379be57f78a4095cd51726495fd"
)

func Convert_MachinesV1alpha1Machine_To_ClusterV1alpha1Machine(in *machinesv1alpha1.Machine, out *clusterv1alpha1.Machine) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec.ObjectMeta = in.Spec.ObjectMeta
	out.SelfLink = ""
	out.UID = ""
	out.ResourceVersion = ""
	out.Generation = 0
	out.CreationTimestamp = metav1.Time{}
	out.ObjectMeta.Namespace = "kube-system"

	// Add annotation that indicates the current revision used for the types
	if out.Annotations == nil {
		out.Annotations = map[string]string{}
	}
	out.Annotations[TypeRevisionAnnotationName] = TypeRevisionCurrentVersion

	// sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1.MachineStatus and
	// pkg/machines/v1alpha1.MachineStatus are semantically identical, the former
	// only has one additional field, so we cast by serializing and deserializing
	inStatusJSON, err := json.Marshal(in.Status)
	if err != nil {
		return fmt.Errorf("failed to marshal downstreammachine status: %v", err)
	}
	if err = json.Unmarshal(inStatusJSON, &out.Status); err != nil {
		return fmt.Errorf("failed to unmarshal downstreammachine status: %v", err)
	}
	out.Spec.ObjectMeta = in.Spec.ObjectMeta
	out.Spec.Taints = in.Spec.Taints
	providerConfigRaw, err := json.Marshal(in.Spec.ProviderConfig)
	if err != nil {
		return err
	}
	out.Spec.ProviderConfig = clusterv1alpha1.ProviderConfig{Value: &runtime.RawExtension{Raw: providerConfigRaw}}

	// This currently results in in.Spec.Versions.ContainerRuntime being dropped,
	// because it was removed from the upstream type in
	// https://github.com/kubernetes-sigs/cluster-api/pull/240
	// To work around this, we put it into the providerConfig
	inMachineVersionJSON, err := json.Marshal(in.Spec.Versions)
	if err != nil {
		return fmt.Errorf("failed to marshal downstreammachine version: %v", err)
	}
	if err = json.Unmarshal(inMachineVersionJSON, &out.Spec.Versions); err != nil {
		return fmt.Errorf("failed to unmarshal downstreammachine version: %v", err)
	}
	out.Spec.ProviderConfig.Value.Raw, err = addContainerRuntimeInfoToProviderConfig(*out.Spec.ProviderConfig.Value,
		in.Spec.Versions.ContainerRuntime)
	if err != nil {
		return fmt.Errorf("failed to add containerRuntimeInfo to providerConfig: %v", err)
	}
	out.Spec.ConfigSource = in.Spec.ConfigSource
	return nil
}

func addContainerRuntimeInfoToProviderConfig(providerConfigValue runtime.RawExtension, containerRuntimeInfo machinesv1alpha1.ContainerRuntimeInfo) ([]byte, error) {
	providerConfigMap := map[string]interface{}{}
	if err := json.Unmarshal(providerConfigValue.Raw, &providerConfigMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshall provider config into map: %v", err)
	}
	// The JSON unmarshall makes the map a null pointer if providerConfigValue.Raw is empty
	if providerConfigMap == nil {
		providerConfigMap = map[string]interface{}{}
	}
	providerConfigMap[ContainerRuntimeInfoKey] = containerRuntimeInfo
	return json.Marshal(providerConfigMap)
}
