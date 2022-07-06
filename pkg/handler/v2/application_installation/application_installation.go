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

package applicationinstallation

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func ListApplicationInstallations(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(listApplicationInstallationsReq)

		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(ctx, userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetClientForCustomerCluster(ctx, userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		installList := &appskubermaticv1.ApplicationInstallationList{}
		if err := client.List(ctx, installList); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		installations := make([]*apiv2.ApplicationInstallation, len(installList.Items))
		for i := range installList.Items {
			installations[i] = convertInternalToExternal(&installList.Items[i])
		}

		return installations, nil
	}
}

func convertInternalToExternal(app *appskubermaticv1.ApplicationInstallation) *apiv2.ApplicationInstallation {
	return &apiv2.ApplicationInstallation{
		ObjectMeta: apiv1.ObjectMeta{
			CreationTimestamp: apiv1.Time(app.CreationTimestamp),
			Name:              app.Name,
		},
		Namespace: app.Namespace,
		Spec:      &app.Spec,
	}
}
