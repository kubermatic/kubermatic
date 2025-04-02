/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package provider

import (
	"k8c.io/machine-controller/sdk/providerconfig"

	"k8s.io/apimachinery/pkg/util/sets"
)

func addTagToSlice(tags []providerconfig.ConfigVarString, tag string) []providerconfig.ConfigVarString {
	if tag == "" {
		return tags
	}

	exists := false
	for _, t := range tags {
		if t.Value == tag {
			exists = true
			break
		}
	}

	if !exists {
		if tags == nil {
			tags = []providerconfig.ConfigVarString{}
		}
		tags = append(tags, providerconfig.ConfigVarString{Value: tag})
	}

	return tags
}

func mergeTags(existing []providerconfig.ConfigVarString, newTags []string) []providerconfig.ConfigVarString {
	tags := sets.New(newTags...)
	for _, tag := range existing {
		tags.Insert(tag.Value)
	}

	existing = []providerconfig.ConfigVarString{}
	for _, tag := range sets.List(tags) {
		existing = append(existing, providerconfig.ConfigVarString{Value: tag})
	}

	return existing
}

func IsConfigVarStringEmpty(val providerconfig.ConfigVarString) bool {
	// Check if SecretKeyRef is empty.
	if val.SecretKeyRef.Namespace != "" ||
		val.SecretKeyRef.Name != "" ||
		val.SecretKeyRef.Key != "" {
		return false
	}
	// Check if ConfigMapKeyRef is empty.
	if val.ConfigMapKeyRef.Namespace != "" ||
		val.ConfigMapKeyRef.Name != "" ||
		val.ConfigMapKeyRef.Key != "" {
		return false
	}
	// Check if Value is empty.
	if val.Value != "" {
		return false
	}
	return true
}
