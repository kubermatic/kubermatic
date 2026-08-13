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

// Package accelerator defines the trusted per-Machine accelerator accounting footprint.
package accelerator

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
	kjson "sigs.k8s.io/json"
)

const (
	// AnnotationPrefix is reserved for KKP-owned Machine accelerator accounting metadata.
	AnnotationPrefix = "accelerators.kubermatic.io/"
	// FootprintAnnotationKey stores the trusted accelerator accounting footprint on a Machine.
	FootprintAnnotationKey = AnnotationPrefix + "footprint"
	// MutatingWebhookPath is the dedicated Machine footprint mutation endpoint.
	MutatingWebhookPath = "/mutate-machine-accelerator-footprint"
	// ValidatingWebhookPath is the dedicated Machine footprint validation endpoint.
	ValidatingWebhookPath = "/validate-machine-accelerator-footprint"

	// SchemaVersionV1Alpha1 is the first version of the footprint annotation payload schema.
	SchemaVersionV1Alpha1 = "v1alpha1"
	// ProviderKubeVirt identifies a footprint derived from KubeVirt Machine intent.
	ProviderKubeVirt = "kubevirt"
)

// Footprint is the versioned accelerator consumption captured for one Machine.
// An empty Resources map records that the Machine was processed and consumes no accelerators.
type Footprint struct {
	SchemaVersion string              `json:"schemaVersion"`
	Provider      string              `json:"provider"`
	Resources     corev1.ResourceList `json:"resources"`
}

// NewKubeVirtFootprint returns an isolated KubeVirt footprint for the given resources.
func NewKubeVirtFootprint(resources corev1.ResourceList) Footprint {
	if resources == nil {
		resources = corev1.ResourceList{}
	} else {
		resources = resources.DeepCopy()
	}

	return Footprint{
		SchemaVersion: SchemaVersionV1Alpha1,
		Provider:      ProviderKubeVirt,
		Resources:     resources,
	}
}

// Encode validates and serializes a footprint into its canonical annotation value.
func Encode(footprint Footprint) (string, error) {
	if err := validate(footprint); err != nil {
		return "", err
	}

	encoded, err := json.Marshal(footprint)
	if err != nil {
		return "", fmt.Errorf("failed to encode accelerator footprint: %w", err)
	}
	return string(encoded), nil
}

// Decode strictly parses and validates an accelerator footprint annotation value.
func Decode(value string) (Footprint, error) {
	footprint := Footprint{}
	strictErrors, err := kjson.UnmarshalStrict([]byte(value), &footprint)
	if err != nil {
		return Footprint{}, fmt.Errorf("failed to decode accelerator footprint: %w", err)
	}
	if len(strictErrors) > 0 {
		messages := make([]string, len(strictErrors))
		for i, strictErr := range strictErrors {
			messages[i] = strictErr.Error()
		}
		sort.Strings(messages)
		return Footprint{}, fmt.Errorf("failed to decode accelerator footprint: %s", strings.Join(messages, "; "))
	}
	if err := validate(footprint); err != nil {
		return Footprint{}, err
	}

	footprint.Resources = footprint.Resources.DeepCopy()
	return footprint, nil
}

// IsReservedAnnotationKey reports whether a key belongs to KKP's accelerator annotation namespace.
func IsReservedAnnotationKey(key string) bool {
	return strings.HasPrefix(key, AnnotationPrefix)
}

func validate(footprint Footprint) error {
	if footprint.SchemaVersion != SchemaVersionV1Alpha1 {
		return fmt.Errorf("unsupported accelerator footprint schemaVersion %q", footprint.SchemaVersion)
	}
	if footprint.Provider != ProviderKubeVirt {
		return fmt.Errorf("unsupported accelerator footprint provider %q", footprint.Provider)
	}
	if footprint.Resources == nil {
		return fmt.Errorf("accelerator footprint resources must not be null")
	}

	resourceNames := make([]string, 0, len(footprint.Resources))
	for resourceName := range footprint.Resources {
		resourceNames = append(resourceNames, string(resourceName))
	}
	sort.Strings(resourceNames)

	for _, resourceName := range resourceNames {
		quantity := footprint.Resources[corev1.ResourceName(resourceName)]
		if strings.Count(resourceName, "/") != 1 {
			return fmt.Errorf("accelerator resource %q must be a qualified name with exactly one '/'", resourceName)
		}
		if validationErrors := k8svalidation.IsQualifiedName(resourceName); len(validationErrors) > 0 {
			return fmt.Errorf("accelerator resource %q is not a valid qualified name: %s", resourceName, strings.Join(validationErrors, "; "))
		}
		if quantity.Sign() <= 0 {
			return fmt.Errorf("accelerator resource %q must have a positive quantity", resourceName)
		}
		if _, exact := quantity.AsScale(0); !exact {
			return fmt.Errorf("accelerator resource %q must have a whole-number quantity", resourceName)
		}
	}

	return nil
}
