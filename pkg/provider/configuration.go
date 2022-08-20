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

package provider

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/defaults"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NonDefaultingKubermaticConfigurationGetter is a function to retrieve the
// currently relevant KubermaticConfiguration. That is the one in the same
// namespace as the running application (e.g. the seed-controller-manager).
// It's an error if there are none or more than one KubermaticConfiguration
// objects in a single namespace.
type KubermaticConfigurationGetter = func(ctx context.Context) (*kubermaticv1.KubermaticConfiguration, error)

// DynamicKubermaticConfigurationGetterFactory returns a dynamic KubermaticConfigurationGetter,
// which will list all Configurations in the given namespace and return the found config or
// an error if 0 or more Configurations where found.
func DynamicKubermaticConfigurationGetterFactory(client ctrlruntimeclient.Reader, namespace string) (KubermaticConfigurationGetter, error) {
	if len(namespace) == 0 {
		return nil, fmt.Errorf("a namespace must be provided")
	}

	return func(ctx context.Context) (*kubermaticv1.KubermaticConfiguration, error) {
		config, err := GetRawKubermaticConfiguration(ctx, client, namespace)
		if err != nil {
			return nil, err
		}

		config, err = defaults.DefaultConfiguration(config, zap.NewNop().Sugar())
		if err != nil {
			return nil, fmt.Errorf("failed to apply default values: %w", err)
		}

		return config, nil
	}, nil
}

var (
	ErrNoKubermaticConfigurationFound      = errors.New("no KubermaticConfiguration resource found")
	ErrTooManyKubermaticConfigurationFound = errors.New("more than one KubermaticConfiguration resource found")
)

// GetRawKubermaticConfiguration will list all Configurations in the given namespace and
// return the found config or an error if 0 or more Configurations where found.
// Most code should use a KubermaticConfigurationGetter instead of calling this function
// directly. This function does not apply the default values.
func GetRawKubermaticConfiguration(ctx context.Context, client ctrlruntimeclient.Reader, namespace string) (*kubermaticv1.KubermaticConfiguration, error) {
	if len(namespace) == 0 {
		return nil, fmt.Errorf("a namespace must be provided")
	}

	configList := kubermaticv1.KubermaticConfigurationList{}
	if err := client.List(ctx, &configList, &ctrlruntimeclient.ListOptions{Namespace: namespace}); err != nil {
		return nil, fmt.Errorf("failed to list KubermaticConfigurations in namespace %q: %w", namespace, err)
	}

	if len(configList.Items) == 0 {
		return nil, ErrNoKubermaticConfigurationFound
	}

	if len(configList.Items) > 1 {
		return nil, ErrTooManyKubermaticConfigurationFound
	}

	return &configList.Items[0], nil
}

// StaticKubermaticConfigurationGetterFactory returns a KubermaticConfigurationGetter that
// returns the same Configuration on every call. This is mostly used for local development
// in order to provide an easy to modify configuration file. Actual production use will use
// the dynamic getter instead.
func StaticKubermaticConfigurationGetterFactory(config *kubermaticv1.KubermaticConfiguration) (KubermaticConfigurationGetter, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}

	return func(ctx context.Context) (*kubermaticv1.KubermaticConfiguration, error) {
		defaulted, err := defaults.DefaultConfiguration(config, zap.NewNop().Sugar())
		if err != nil {
			return nil, fmt.Errorf("failed to apply default values: %w", err)
		}

		return defaulted, nil
	}, nil
}
