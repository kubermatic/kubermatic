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

package constrainttemplate

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func ListEndpoint(userInfoGetter provider.UserInfoGetter, constraintTemplateProvider provider.ConstraintTemplateProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		constraintTemplateList, err := constraintTemplateProvider.List()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		apiCT := make([]*apiv2.ConstraintTemplate, 0)
		for _, ct := range constraintTemplateList.Items {
			apiCT = append(apiCT, convertCTToAPI(&ct))
		}

		return apiCT, nil
	}
}

func convertCTToAPI(ct *v1beta1.ConstraintTemplate) *apiv2.ConstraintTemplate {
	return &apiv2.ConstraintTemplate{
		Name:   ct.Name,
		Spec:   ct.Spec,
		Status: ct.Status,
	}
}
