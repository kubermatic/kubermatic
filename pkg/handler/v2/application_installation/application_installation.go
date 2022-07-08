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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func ListApplicationInstallations(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(listApplicationInstallationsReq)

		client, err := userClusterClientFromContext(ctx, userInfoGetter, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
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

func CreateApplicationInstallation(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(createApplicationInstallationReq)

		client, err := userClusterClientFromContext(ctx, userInfoGetter, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		internalAppInstall := convertExternalToInternal(&req.Body)
		if err := client.Create(ctx, internalAppInstall); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalToExternal(internalAppInstall), nil
	}
}

func DeleteApplicationInstallation(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(deleteApplicationInstallationReq)

		client, err := userClusterClientFromContext(ctx, userInfoGetter, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		delObj := &appskubermaticv1.ApplicationInstallation{
			TypeMeta: metav1.TypeMeta{
				Kind:       appskubermaticv1.ApplicationInstallationKindName,
				APIVersion: appskubermaticv1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.ApplicationInstallationName,
				Namespace: req.Namespace,
			},
		}
		err = client.Delete(ctx, delObj)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return nil, nil
	}
}

func GetApplicationInstallation(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(getApplicationInstallationReq)

		client, err := userClusterClientFromContext(ctx, userInfoGetter, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		applicationInstallation := &appskubermaticv1.ApplicationInstallation{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.ApplicationInstallationName}, applicationInstallation); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalToExternal(applicationInstallation), nil
	}
}

func UpdateApplicationInstallation(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(updateApplicationInstallationReq)

		client, err := userClusterClientFromContext(ctx, userInfoGetter, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		// first fetch the currentAppInstall to make sure it exists
		currentAppInstall := &appskubermaticv1.ApplicationInstallation{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.ApplicationInstallationName}, currentAppInstall); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		newAppInstall := convertExternalToInternal(&req.Body)
		currentAppInstall.Spec = newAppInstall.Spec

		err = client.Update(ctx, currentAppInstall)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalToExternal(currentAppInstall), nil
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

func convertExternalToInternal(app *apiv2.ApplicationInstallation) *appskubermaticv1.ApplicationInstallation {
	return &appskubermaticv1.ApplicationInstallation{
		TypeMeta: metav1.TypeMeta{
			Kind:       appskubermaticv1.ApplicationInstallationKindName,
			APIVersion: appskubermaticv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
		Spec: *app.Spec,
	}
}

func userClusterClientFromContext(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID, clusterID string) (ctrlruntimeclient.Client, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	cluster, err := clusterProvider.Get(ctx, userInfo, clusterID, &provider.ClusterGetOptions{})
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	client, err := clusterProvider.GetClientForUserCluster(ctx, userInfo, cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return client, nil
}
