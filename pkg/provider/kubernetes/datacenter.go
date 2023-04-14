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

package kubernetes

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/provider"

	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// DatacenterGetterFactory returns a DatacenterGetter. It has validation of all its arguments.
func DatacenterGetterFactory(client ctrlruntimeclient.Reader) (provider.DatacenterGetter, error) {
	return func(ctx context.Context, dcName string) (*kubermaticv1.Datacenter, error) {
		Datacenter := &kubermaticv1.Datacenter{}
		if err := client.Get(ctx, types.NamespacedName{Name: dcName}, Datacenter); err != nil {
			return nil, fmt.Errorf("failed to get datacenter %q: %w", dcName, err)
		}

		return Datacenter, nil
	}, nil
}

func DatacentersGetterFactory(client ctrlruntimeclient.Client) (provider.DatacentersGetter, error) {
	return func(ctx context.Context) (map[string]*kubermaticv1.Datacenter, error) {
		list := &kubermaticv1.DatacenterList{}
		if err := client.List(ctx, list); err != nil {
			return nil, fmt.Errorf("failed to list datacenters: %w", err)
		}

		result := map[string]*kubermaticv1.Datacenter{}
		for i, item := range list.Items {
			result[item.Name] = &list.Items[i]
		}

		return result, nil
	}, nil
}
