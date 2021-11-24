/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package helper

import (
	"fmt"
	"reflect"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
)

func getProviderValue(s *kubermaticv1.PresetSpec, providerType kubermaticv1.ProviderType) reflect.Value {
	spec := reflect.ValueOf(s).Elem()
	if spec.Kind() != reflect.Struct {
		return reflect.Value{}
	}

	// BYO cloud provider is generally not supported in Presets, but it wouldn't hurt
	// to call this function with BringYourOwnCloudProvider, as we simply return a
	// zero value and all other functions will then just ignore it.

	ignoreCaseCompare := func(name string) bool {
		return strings.EqualFold(name, string(providerType))
	}

	provider := reflect.Indirect(spec).FieldByNameFunc(ignoreCaseCompare)
	return provider
}

func SetProviderEnabled(p *kubermaticv1.Preset, providerType kubermaticv1.ProviderType, enabled bool) {
	provider := getProviderValue(&p.Spec, providerType)

	ignoreCaseCompare := func(name string) bool {
		return strings.EqualFold(name, "Enabled")
	}

	enabledField := reflect.Indirect(provider).FieldByNameFunc(ignoreCaseCompare)
	enabledField.Set(reflect.ValueOf(&enabled))
}

func HasProvider(p *kubermaticv1.Preset, providerType kubermaticv1.ProviderType) (bool, reflect.Value) {
	provider := getProviderValue(&p.Spec, providerType)
	return !provider.IsZero(), provider
}

func GetProviderList(p *kubermaticv1.Preset) []kubermaticv1.ProviderType {
	existingProviders := []kubermaticv1.ProviderType{}
	for _, provType := range kubermaticv1.SupportedProviders {
		if hasProvider, _ := HasProvider(p, provType); hasProvider {
			existingProviders = append(existingProviders, provType)
		}
	}

	return existingProviders
}

func GetProviderPreset(p *kubermaticv1.Preset, providerType kubermaticv1.ProviderType) *kubermaticv1.ProviderPreset {
	hasProvider, providerField := HasProvider(p, providerType)
	if !hasProvider {
		return nil
	}

	presetSpecBaseFieldName := "ProviderPreset"
	presetBaseField := reflect.Indirect(providerField).FieldByName(presetSpecBaseFieldName)
	presetBase := presetBaseField.Interface().(kubermaticv1.ProviderPreset)
	return &presetBase
}

func Validate(p *kubermaticv1.Preset, providerType kubermaticv1.ProviderType) error {
	hasProvider, providerField := HasProvider(p, providerType)
	if !hasProvider {
		return fmt.Errorf("missing provider configuration for: %s", providerType)
	}

	type validateable interface {
		IsValid() bool
	}

	validateableType := reflect.TypeOf(new(validateable)).Elem()
	if !providerField.Type().Implements(validateableType) {
		return fmt.Errorf("provider %s does not implement validateable interface", providerField.Type().Name())
	}

	checker := providerField.Interface().(validateable)
	if !checker.IsValid() {
		return fmt.Errorf("required fields missing for provider spec: %s", providerType)
	}

	return nil
}

func IsProviderEnabled(p *kubermaticv1.Preset, provider kubermaticv1.ProviderType) bool {
	presetProvider := GetProviderPreset(p, provider)
	return presetProvider != nil && presetProvider.IsEnabled()
}

func OverrideProvider(p *kubermaticv1.Preset, providerType kubermaticv1.ProviderType, newPreset *kubermaticv1.Preset) *kubermaticv1.Preset {
	dest := getProviderValue(&p.Spec, providerType)
	src := getProviderValue(&newPreset.Spec, providerType)
	dest.Set(src)

	return p
}

func RemoveProvider(p *kubermaticv1.Preset, providerType kubermaticv1.ProviderType) *kubermaticv1.Preset {
	provider := getProviderValue(&p.Spec, providerType)
	provider.Set(reflect.Zero(provider.Type()))

	return p
}
