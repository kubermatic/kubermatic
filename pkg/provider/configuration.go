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
	"fmt"

	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// KubermaticConfigurationGetter is a function to retrieve the currently relevant
// KubermaticConfiguration. That is the one in the same namespace as the
// running application (e.g. the seed-controller-manager). It's an error
// if there are none or more than one KubermaticConfiguration objects in
// a single namespace.
type KubermaticConfigurationGetter = func(ctx context.Context) (*operatorv1alpha1.KubermaticConfiguration, error)

// KubermaticConfigurationGetterFactory returns a KubermaticConfigurationGetter.
func KubermaticConfigurationGetterFactory(client ctrlruntimeclient.Reader, namespace string) (KubermaticConfigurationGetter, error) {
	if len(namespace) == 0 {
		return nil, fmt.Errorf("a namespace must be provided")
	}

	return func(ctx context.Context) (*operatorv1alpha1.KubermaticConfiguration, error) {
		configList := operatorv1alpha1.KubermaticConfigurationList{}
		if err := client.List(ctx, &configList, &ctrlruntimeclient.ListOptions{Namespace: namespace}); err != nil {
			return nil, fmt.Errorf("failed to list KubermaticConfigurations in namespace %q: %w", namespace, err)
		}

		if len(configList.Items) == 0 {
			return nil, fmt.Errorf("no KubermaticConfiguration resource found in namespace %q", namespace)
		}

		if len(configList.Items) > 1 {
			return nil, fmt.Errorf("more than one KubermaticConfiguration resource found in namespace %q", namespace)
		}

		return &configList.Items[0], nil
	}, nil
}
