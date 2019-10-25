/*
Copyright 2019 The Machine Controller Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubevirt

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdi "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"

	"github.com/kubermatic/machine-controller/pkg/apis/cluster/common"
	"github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	cloudprovidererrors "github.com/kubermatic/machine-controller/pkg/cloudprovider/errors"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/instance"
	cloudprovidertypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/types"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	utilpointer "k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	// Workaround until we have https://github.com/kubevirt/kubevirt/pull/2841
	metav1.AddToGroupVersion(scheme.Scheme, kubevirtv1.GroupVersion)
}

type provider struct {
	configVarResolver *providerconfig.ConfigVarResolver
}

// New returns a Kubevirt provider
func New(configVarResolver *providerconfig.ConfigVarResolver) cloudprovidertypes.Provider {
	return &provider{configVarResolver: configVarResolver}
}

type RawConfig struct {
	Kubeconfig       providerconfig.ConfigVarString `json:"kubeconfig,omitempty"`
	CPUs             providerconfig.ConfigVarString `json:"cpus,omitempty"`
	Memory           providerconfig.ConfigVarString `json:"memory,omitempty"`
	Namespace        providerconfig.ConfigVarString `json:"namespace,omitempty"`
	SourceURL        providerconfig.ConfigVarString `json:"sourceURL,omitempty"`
	PVCSize          providerconfig.ConfigVarString `json:"pvcSize,omitempty"`
	StorageClassName providerconfig.ConfigVarString `json:"storageClassName,omitempty"`
}

type Config struct {
	Kubeconfig       rest.Config
	CPUs             string
	Memory           string
	Namespace        string
	SourceURL        string
	StorageClassName string
	PVCSize          resource.Quantity
}

type kubeVirtServer struct {
	vmi kubevirtv1.VirtualMachineInstance
}

func (k *kubeVirtServer) Name() string {
	return k.vmi.Name
}

func (k *kubeVirtServer) ID() string {
	return string(k.vmi.UID)
}

func (k *kubeVirtServer) Addresses() []string {
	var addresses []string
	for _, kvInterface := range k.vmi.Status.Interfaces {
		if address := strings.Split(kvInterface.IP, "/")[0]; address != "" {
			addresses = append(addresses, address)
		}
	}
	return addresses
}

func (k *kubeVirtServer) Status() instance.Status {
	if k.vmi.Status.Phase == kubevirtv1.Running {
		return instance.StatusRunning
	}
	return instance.StatusUnknown
}

var _ instance.Instance = &kubeVirtServer{}

func (p *provider) getConfig(s v1alpha1.ProviderSpec) (*Config, *providerconfig.Config, error) {
	if s.Value == nil {
		return nil, nil, fmt.Errorf("machine.spec.providerconfig.value is nil")
	}
	pconfig := providerconfig.Config{}
	err := json.Unmarshal(s.Value.Raw, &pconfig)
	if err != nil {
		return nil, nil, err
	}

	//TODO: Use RawConfig to allow resolving via secretReg/ConfigMapRef
	rawConfig := RawConfig{}
	if err = json.Unmarshal(pconfig.CloudProviderSpec.Raw, &rawConfig); err != nil {
		return nil, nil, err
	}
	config := Config{}
	configString, err := p.configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.Kubeconfig, "KUBEVIRT_KUBECONFIG")
	if err != nil {
		return nil, nil, fmt.Errorf(`failed to get value of "config" field: %v`, err)
	}
	config.CPUs, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.CPUs)
	if err != nil {
		return nil, nil, fmt.Errorf(`failed to get value of "cpus" field: %v`, err)
	}
	config.Memory, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Memory)
	if err != nil {
		return nil, nil, fmt.Errorf(`failed to get value of "memory" field: %v`, err)
	}
	config.Namespace, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Namespace)
	if err != nil {
		return nil, nil, fmt.Errorf(`failed to get value of "namespace" field: %v`, err)
	}
	config.SourceURL, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.SourceURL)
	if err != nil {
		return nil, nil, fmt.Errorf(`failed to get value of "sourceURL" field: %v`, err)
	}
	pvcSize, err := p.configVarResolver.GetConfigVarStringValue(rawConfig.PVCSize)
	if err != nil {
		return nil, nil, fmt.Errorf(`failed to get value of "pvcSize" field: %v`, err)
	}
	if config.PVCSize, err = resource.ParseQuantity(pvcSize); err != nil {
		return nil, nil, fmt.Errorf(`failed to parse value of "pvcSize" field: %v`, err)
	}
	config.StorageClassName, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.StorageClassName)
	if err != nil {
		return nil, nil, fmt.Errorf(`failed to get value of "storageClassName" field: %v`, err)
	}
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(configString))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode kubeconfig: %v", err)
	}
	config.Kubeconfig = *restConfig

	return &config, &pconfig, nil
}

func (p *provider) Get(machine *v1alpha1.Machine, _ *cloudprovidertypes.ProviderData) (instance.Instance, error) {
	c, _, err := p.getConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Failed to parse MachineSpec, due to %v", err),
		}
	}
	sigClient, err := client.New(&c.Kubeconfig, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to get kubevirt client: %v", err)
	}
	ctx := context.Background()

	virtualMachine := &kubevirtv1.VirtualMachine{}
	if err := sigClient.Get(ctx, types.NamespacedName{Namespace: c.Namespace, Name: machine.Name}, virtualMachine); err != nil {
		if !kerrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to get VirtualMachine %s: %v", machine.Name, err)
		}
		return nil, cloudprovidererrors.ErrInstanceNotFound
	}

	virtualMachineInstance := &kubevirtv1.VirtualMachineInstance{}
	if err := sigClient.Get(ctx, types.NamespacedName{Namespace: c.Namespace, Name: machine.Name}, virtualMachineInstance); err != nil {
		if kerrors.IsNotFound(err) {
			return &kubeVirtServer{}, nil
		}

		return nil, err
	}

	// Deletion takes some time, so consider the VMI as deleted as soon as it has a DeletionTimestamp
	// because once the node got into status not ready its informers wont fire again
	// With the current approach we may run into a conflict when creating the VMI again, however this
	// results in the machine being reqeued
	if virtualMachineInstance.DeletionTimestamp != nil {
		return nil, cloudprovidererrors.ErrInstanceNotFound
	}

	if virtualMachineInstance.Status.Phase == kubevirtv1.Failed ||
		// The VMI enters phase succeeded if someone issues a kubectl
		// delete pod on the virt-launcher pod it runs in
		virtualMachineInstance.Status.Phase == kubevirtv1.Succeeded {
		// The pod got deleted, delete the VMI and return ErrNotFound so the VMI
		// will get recreated
		if err := sigClient.Delete(ctx, virtualMachineInstance); err != nil {
			return nil, fmt.Errorf("failed to delete failed VMI %s: %v", machine.Name, err)
		}
		return nil, cloudprovidererrors.ErrInstanceNotFound
	}

	return &kubeVirtServer{vmi: *virtualMachineInstance}, nil
}

// We don't use the UID for kubevirt because the name of a VMI must stay stable
// in order for the node name to stay stable. The operator is responsible for ensuring
// there are no conflicts, e.G. by using one Namespace per Kubevirt user cluster
func (p *provider) MigrateUID(machine *v1alpha1.Machine, new types.UID) error {
	return nil
}

func (p *provider) Validate(spec v1alpha1.MachineSpec) error {
	c, pc, err := p.getConfig(spec.ProviderSpec)
	if err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}
	if _, err := parseResources(c.CPUs, c.Memory); err != nil {
		return err
	}
	if pc.OperatingSystem == providerconfig.OperatingSystemCoreos {
		return fmt.Errorf("CoreOS is not supported")
	}
	sigClient, err := client.New(&c.Kubeconfig, client.Options{})
	if err != nil {
		return fmt.Errorf("failed to get kubevirt client: %v", err)
	}
	// Check if we can reach the API of the target cluster
	vmi := &kubevirtv1.VirtualMachineInstance{}
	if err := sigClient.Get(context.Background(), types.NamespacedName{Namespace: c.Namespace, Name: "not-expected-to-exist"}, vmi); err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("failed to request VirtualMachineInstances: %v", err)
	}

	return nil
}

func (p *provider) AddDefaults(spec v1alpha1.MachineSpec) (v1alpha1.MachineSpec, error) {
	return spec, nil
}

func (p *provider) GetCloudConfig(spec v1alpha1.MachineSpec) (config string, name string, err error) {
	return "", "", nil
}

func (p *provider) MachineMetricsLabels(machine *v1alpha1.Machine) (map[string]string, error) {
	labels := make(map[string]string)

	c, _, err := p.getConfig(machine.Spec.ProviderSpec)
	if err == nil {
		labels["cpus"] = c.CPUs
		labels["memoryMIB"] = c.Memory
		labels["sourceURL"] = c.SourceURL
	}

	return labels, err
}

func (p *provider) Create(machine *v1alpha1.Machine, _ *cloudprovidertypes.ProviderData, userdata string) (instance.Instance, error) {
	c, _, err := p.getConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Failed to parse MachineSpec, due to %v", err),
		}
	}

	// We add the timestamp because the secret name must be different when we recreate the VMI
	// because its pod got deleted
	// The secret has an ownerRef on the VMI so garbace collection will take care of cleaning up
	terminationGracePeriodSeconds := int64(30)
	userDataSecretName := fmt.Sprintf("userdata-%s-%s", machine.Name, strconv.Itoa(int(time.Now().Unix())))
	requestsAndLimits, err := parseResources(c.CPUs, c.Memory)
	if err != nil {
		return nil, err
	}

	var (
		pvcRequest     = corev1.ResourceList{corev1.ResourceStorage: c.PVCSize}
		dataVolumeName = machine.Name
	)

	// we need this check until this issue is resolved:
	// https://github.com/kubevirt/containerized-data-importer/issues/895
	if len(dataVolumeName) > 63 {
		return nil, fmt.Errorf("dataVolumeName size %v, is bigger than 63 characters", len(dataVolumeName))
	}

	virtualMachine := &kubevirtv1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      machine.Name,
			Namespace: c.Namespace,
			Labels: map[string]string{
				"kubevirt.io/vm": machine.Name,
			},
		},
		Spec: kubevirtv1.VirtualMachineSpec{
			Running: utilpointer.BoolPtr(true),
			Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kubevirt.io/vm": machine.Name,
					},
				},
				Spec: kubevirtv1.VirtualMachineInstanceSpec{
					Domain: kubevirtv1.DomainSpec{
						CPU: &kubevirtv1.CPU{
							Cores: 2,
						},
						Devices: kubevirtv1.Devices{
							Disks: []kubevirtv1.Disk{
								{
									Name:       "datavolumedisk",
									DiskDevice: kubevirtv1.DiskDevice{Disk: &kubevirtv1.DiskTarget{Bus: "virtio"}},
								},
								{
									Name:       "cloudinitdisk",
									DiskDevice: kubevirtv1.DiskDevice{Disk: &kubevirtv1.DiskTarget{Bus: "virtio"}},
								},
							},
						},
						Resources: kubevirtv1.ResourceRequirements{
							Requests: *requestsAndLimits,
							Limits:   *requestsAndLimits,
						},
					},
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					Volumes: []kubevirtv1.Volume{
						{
							Name: "datavolumedisk",
							VolumeSource: kubevirtv1.VolumeSource{
								DataVolume: &kubevirtv1.DataVolumeSource{
									Name: dataVolumeName,
								},
							},
						},
						{
							Name: "cloudinitdisk",
							VolumeSource: kubevirtv1.VolumeSource{
								CloudInitNoCloud: &kubevirtv1.CloudInitNoCloudSource{
									UserDataSecretRef: &corev1.LocalObjectReference{
										Name: userDataSecretName,
									},
								},
							},
						},
					},
				},
			},
			DataVolumeTemplates: []cdi.DataVolume{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: dataVolumeName,
					},
					Spec: cdi.DataVolumeSpec{
						PVC: &corev1.PersistentVolumeClaimSpec{
							StorageClassName: utilpointer.StringPtr(c.StorageClassName),
							AccessModes: []corev1.PersistentVolumeAccessMode{
								"ReadWriteOnce",
							},
							Resources: corev1.ResourceRequirements{
								Requests: pvcRequest,
							},
						},
						Source: cdi.DataVolumeSource{
							HTTP: &cdi.DataVolumeSourceHTTP{
								URL: c.SourceURL,
							},
						},
					},
				},
			},
		},
	}

	sigClient, err := client.New(&c.Kubeconfig, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to get kubevirt client: %v", err)
	}
	ctx := context.Background()

	if err := sigClient.Create(ctx, virtualMachine); err != nil {
		return nil, fmt.Errorf("failed to create vmi: %v", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            userDataSecretName,
			Namespace:       virtualMachine.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(virtualMachine, kubevirtv1.VirtualMachineGroupVersionKind)},
		},
		Data: map[string][]byte{"userdata": []byte(userdata)},
	}
	if err := sigClient.Create(ctx, secret); err != nil {
		return nil, fmt.Errorf("failed to create secret for userdata: %v", err)
	}
	return &kubeVirtServer{}, nil

}

func (p *provider) Cleanup(machine *v1alpha1.Machine, _ *cloudprovidertypes.ProviderData) (bool, error) {
	c, _, err := p.getConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return false, cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Failed to parse MachineSpec, due to %v", err),
		}
	}
	sigClient, err := client.New(&c.Kubeconfig, client.Options{})
	if err != nil {
		return false, fmt.Errorf("failed to get kubevirt client: %v", err)
	}
	ctx := context.Background()

	vm := &kubevirtv1.VirtualMachine{}
	if err := sigClient.Get(ctx, types.NamespacedName{Namespace: c.Namespace, Name: machine.Name}, vm); err != nil {
		if !kerrors.IsNotFound(err) {
			return false, fmt.Errorf("failed to get VirtualMachineInstance %s: %v", machine.Name, err)
		}
		// VMI is gone
		return true, nil
	}

	return false, sigClient.Delete(ctx, vm)
}

func parseResources(cpus, memory string) (*corev1.ResourceList, error) {
	memoryResource, err := resource.ParseQuantity(memory)
	if err != nil {
		return nil, fmt.Errorf("failed to parse memory requests: %v", err)
	}
	cpuResource, err := resource.ParseQuantity(cpus)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cpu request: %v", err)
	}
	return &corev1.ResourceList{
		corev1.ResourceMemory: memoryResource,
		corev1.ResourceCPU:    cpuResource,
	}, nil
}

func (p *provider) SetMetricsForMachines(machines v1alpha1.MachineList) error {
	return nil
}
