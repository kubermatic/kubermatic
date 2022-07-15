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

package applicationdefinition

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func ListApplicationDefinitions(applicationDefinitionProvider provider.ApplicationDefinitionProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		defList, err := applicationDefinitionProvider.ListUnsecured(ctx)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		definitions := make([]*apiv2.ApplicationDefinition, len(defList.Items))
		for i := range defList.Items {
			definitions[i] = convertInternalToExternal(&defList.Items[i])
		}

		return definitions, nil
	}
}

func convertInternalToExternal(appDef *appskubermaticv1.ApplicationDefinition) *apiv2.ApplicationDefinition {
	return &apiv2.ApplicationDefinition{
		ObjectMeta: apiv1.ObjectMeta{
			CreationTimestamp: apiv1.Time(appDef.CreationTimestamp),
			Name:              appDef.Name,
		},
		Spec: &appDef.Spec,
	}
}
