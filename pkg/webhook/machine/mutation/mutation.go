/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

// Package mutation captures the trusted accelerator footprint of KubeVirt Machines.
package mutation

import (
	"context"
	"errors"
	"fmt"

	kubevirtv1 "kubevirt.io/api/core/v1"

	"k8c.io/kubermatic/v2/pkg/machine/accelerator"
	kubevirtprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	kubevirttypes "k8c.io/machine-controller/sdk/cloudprovider/kubevirt"
	"k8c.io/machine-controller/sdk/providerconfig"
	"k8c.io/machine-controller/sdk/providerconfig/configvar"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const envKubeVirtKubeconfig = "KUBEVIRT_KUBECONFIG"

type instanceTypeResolver func(context.Context, string, string, *kubevirtv1.InstancetypeMatcher) (*kubevirtprovider.ResolvedInstanceType, error)

// Mutator stamps KKP-owned accelerator accounting metadata on KubeVirt Machines.
type Mutator struct {
	userClient             ctrlruntimeclient.Client
	kubeVirtInfraNamespace string
	resolveInstanceType    instanceTypeResolver
}

// NewMutator returns a Machine accelerator footprint mutator.
func NewMutator(userClient ctrlruntimeclient.Client, kubeVirtInfraNamespace string) *Mutator {
	return &Mutator{
		userClient:             userClient,
		kubeVirtInfraNamespace: kubeVirtInfraNamespace,
		resolveInstanceType:    kubevirtprovider.ResolveInstanceType,
	}
}

var _ admission.Defaulter[*clusterv1alpha1.Machine] = &Mutator{}

// Default recomputes the trusted footprint from Machine intent. User-supplied values under
// the reserved annotation prefix are removed before the server-owned value is written.
func (m *Mutator) Default(ctx context.Context, machine *clusterv1alpha1.Machine) error {
	removeReservedAnnotations(machine)

	config, err := providerconfig.GetConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return fmt.Errorf("failed to read machine.spec.providerSpec: %w", err)
	}
	if config.CloudProvider != providerconfig.CloudProviderKubeVirt {
		return nil
	}

	rawConfig, err := kubevirttypes.GetConfig(*config)
	if err != nil {
		return fmt.Errorf("failed to get KubeVirt provider configuration: %w", err)
	}

	footprint := accelerator.NewKubeVirtFootprint(nil)
	matcher := rawConfig.VirtualMachine.Instancetype
	if matcher != nil {
		if err := validateInstanceTypeMatcher(matcher); err != nil {
			return err
		}
		if m.kubeVirtInfraNamespace == "" {
			return fmt.Errorf("cannot resolve KubeVirt instancetype %q: no infrastructure namespace configured", matcher.Name)
		}

		kubeconfig, err := configvar.NewResolver(ctx, m.userClient).GetStringValueOrEnv(rawConfig.Auth.Kubeconfig, envKubeVirtKubeconfig)
		if err != nil {
			return fmt.Errorf("failed to resolve KubeVirt kubeconfig: %w", err)
		}
		if kubeconfig == "" {
			return fmt.Errorf("failed to resolve KubeVirt kubeconfig: value is empty")
		}

		resolved, err := m.resolveInstanceType(ctx, kubeconfig, m.kubeVirtInfraNamespace, matcher)
		if err != nil {
			return fmt.Errorf("failed to resolve KubeVirt instancetype %q: %w", matcher.Name, err)
		}
		if resolved == nil {
			return fmt.Errorf("failed to resolve KubeVirt instancetype %q: resolver returned no result", matcher.Name)
		}
		footprint = accelerator.NewKubeVirtFootprint(resolved.DeviceResources)
	}

	encodedFootprint, err := accelerator.Encode(footprint)
	if err != nil {
		return fmt.Errorf("failed to encode Machine accelerator footprint: %w", err)
	}
	if machine.Annotations == nil {
		machine.Annotations = map[string]string{}
	}
	machine.Annotations[accelerator.FootprintAnnotationKey] = encodedFootprint

	return nil
}

func validateInstanceTypeMatcher(matcher *kubevirtv1.InstancetypeMatcher) error {
	if matcher.RevisionName != "" {
		return errors.New("KubeVirt instancetype revisionName is not supported by accelerator accounting")
	}
	if matcher.InferFromVolume != "" {
		return errors.New("KubeVirt instancetype inference from volume is not supported by accelerator accounting")
	}
	if matcher.InferFromVolumeFailurePolicy != nil {
		return errors.New("KubeVirt instancetype inference failure policy is not supported by accelerator accounting")
	}
	if matcher.Name == "" {
		return errors.New("KubeVirt instancetype matcher name must not be empty")
	}
	return nil
}

func removeReservedAnnotations(machine *clusterv1alpha1.Machine) {
	for key := range machine.Annotations {
		if accelerator.IsReservedAnnotationKey(key) {
			delete(machine.Annotations, key)
		}
	}
	if len(machine.Annotations) == 0 {
		machine.Annotations = nil
	}
}
