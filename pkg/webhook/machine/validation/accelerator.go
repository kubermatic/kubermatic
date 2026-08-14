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

package validation

import (
	"context"
	"fmt"
	"maps"
	"sort"

	"k8c.io/kubermatic/v2/pkg/machine/accelerator"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	"k8c.io/machine-controller/sdk/providerconfig"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// acceleratorValidator validates the trusted accelerator accounting contract on Machines.
type acceleratorValidator struct{}

// NewAcceleratorValidator returns a Machine accelerator accounting validator.
func NewAcceleratorValidator() *acceleratorValidator {
	return &acceleratorValidator{}
}

var _ admission.Validator[*clusterv1alpha1.Machine] = &acceleratorValidator{}

// ValidateCreate ensures that only the KKP-generated KubeVirt footprint is stored under the
// reserved accelerator annotation prefix.
func (v *acceleratorValidator) ValidateCreate(_ context.Context, machine *clusterv1alpha1.Machine) (admission.Warnings, error) {
	config, err := providerconfig.GetConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to read machine.spec.providerSpec while validating accelerator footprint: %w", err)
	}

	reserved := reservedAnnotations(machine.Annotations)
	reservedKeys := make([]string, 0, len(reserved))
	for key := range reserved {
		reservedKeys = append(reservedKeys, key)
	}
	sort.Strings(reservedKeys)
	for _, key := range reservedKeys {
		if key != accelerator.FootprintAnnotationKey {
			return nil, fmt.Errorf("Machine annotation %q uses KKP's reserved accelerator prefix", key)
		}
	}

	footprintValue, hasFootprint := reserved[accelerator.FootprintAnnotationKey]
	if config.CloudProvider != providerconfig.CloudProviderKubeVirt {
		if hasFootprint {
			return nil, fmt.Errorf("Machine annotation %q is only valid for KubeVirt Machines", accelerator.FootprintAnnotationKey)
		}
		return nil, nil
	}

	if !hasFootprint {
		return nil, fmt.Errorf("KubeVirt Machine is missing KKP-managed annotation %q", accelerator.FootprintAnnotationKey)
	}
	if _, err := accelerator.Decode(footprintValue); err != nil {
		return nil, fmt.Errorf("Machine annotation %q is invalid: %w", accelerator.FootprintAnnotationKey, err)
	}

	return nil, nil
}

// ValidateUpdate keeps the footprint and its source Machine intent immutable, except for
// machine-controller's one-time migration from the removed providerConfig field.
func (v *acceleratorValidator) ValidateUpdate(_ context.Context, oldMachine, newMachine *clusterv1alpha1.Machine) (admission.Warnings, error) {
	oldReserved := reservedAnnotations(oldMachine.Annotations)
	newReserved := reservedAnnotations(newMachine.Annotations)
	if !maps.Equal(oldReserved, newReserved) {
		return nil, fmt.Errorf("Machine annotations under %q are managed by KKP and are immutable", accelerator.AnnotationPrefix)
	}

	if apiequality.Semantic.DeepEqual(oldMachine.Spec.ProviderSpec, newMachine.Spec.ProviderSpec) {
		return nil, nil
	}
	if isLegacyProviderConfigMigration(oldMachine, newMachine, oldReserved, newReserved) {
		return nil, nil
	}

	oldConfig, oldErr := providerconfig.GetConfig(oldMachine.Spec.ProviderSpec)
	newConfig, newErr := providerconfig.GetConfig(newMachine.Spec.ProviderSpec)
	if len(oldReserved) > 0 || len(newReserved) > 0 || oldErr != nil || newErr != nil ||
		oldConfig.CloudProvider == providerconfig.CloudProviderKubeVirt ||
		newConfig.CloudProvider == providerconfig.CloudProviderKubeVirt {
		return nil, fmt.Errorf("Machine spec.providerSpec is immutable for accelerator accounting")
	}

	return nil, nil
}

func isLegacyProviderConfigMigration(oldMachine, newMachine *clusterv1alpha1.Machine, oldReserved, newReserved map[string]string) bool {
	if len(oldReserved) != 0 || len(newReserved) != 0 {
		return false
	}

	oldProviderSpec := oldMachine.Spec.ProviderSpec
	if oldProviderSpec.Value != nil || oldProviderSpec.ValueFrom != nil {
		return false
	}

	newProviderSpec := newMachine.Spec.ProviderSpec
	if newProviderSpec.Value == nil || len(newProviderSpec.Value.Raw) == 0 || newProviderSpec.ValueFrom != nil {
		return false
	}

	// machine-controller validates its one-shot providerConfig migration bypass in a
	// mutating webhook and removes the annotation before validating webhooks run. The
	// empty-to-valid ProviderSpec transition is therefore the remaining migration shape.
	// It does not add a footprint; accounting readiness continues to classify the Machine
	// as legacy until it is recreated through CREATE admission.
	_, err := providerconfig.GetConfig(newProviderSpec)
	return err == nil
}

func (v *acceleratorValidator) ValidateDelete(_ context.Context, _ *clusterv1alpha1.Machine) (admission.Warnings, error) {
	return nil, nil
}

func reservedAnnotations(annotations map[string]string) map[string]string {
	reserved := map[string]string{}
	for key, value := range annotations {
		if accelerator.IsReservedAnnotationKey(key) {
			reserved[key] = value
		}
	}
	return reserved
}
