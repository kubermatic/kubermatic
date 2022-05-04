/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package seed

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
)

// ListSeedNamesEndpoint returns a list of all seed names.
func ListSeedNamesEndpoint(seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		seedMap, err := seedsGetter()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		var resultList []string

		for key := range seedMap {
			resultList = append(resultList, key)
		}

		return resultList, nil
	}
}
