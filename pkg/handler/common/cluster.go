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

package common

import (
	"context"
	"errors"
	"net/http"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func GetInternalCluster(ctx context.Context, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ClusterProvider, privilegedClusterProvider provider.PrivilegedClusterProvider, project *kubermaticv1.Project, projectID, clusterID string, options *provider.ClusterGetOptions) (*kubermaticv1.Cluster, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, utilerrors.NewFromKubernetesError(err)
	}
	if adminUserInfo.IsAdmin {
		cluster, err := privilegedClusterProvider.GetUnsecured(ctx, project, clusterID, options)
		if err != nil {
			return nil, utilerrors.NewFromKubernetesError(err)
		}
		return cluster, nil
	}

	return getClusterForRegularUser(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, projectID, clusterID, options)
}

func getClusterForRegularUser(ctx context.Context, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ClusterProvider, privilegedClusterProvider provider.PrivilegedClusterProvider, project *kubermaticv1.Project, projectID, clusterID string, options *provider.ClusterGetOptions) (*kubermaticv1.Cluster, error) {
	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, utilerrors.NewFromKubernetesError(err)
	}
	cluster, err := clusterProvider.Get(ctx, userInfo, clusterID, options)
	if err != nil {
		// Request came from the specified user. Instead `Not found` error status the `Forbidden` is returned.
		// Next request with privileged user checks if the cluster doesn't exist or some other error occurred.
		if !isStatus(err, http.StatusForbidden) {
			return nil, utilerrors.NewFromKubernetesError(err)
		}
		// Check if cluster really doesn't exist or some other error occurred.
		if _, errGetUnsecured := privilegedClusterProvider.GetUnsecured(ctx, project, clusterID, options); errGetUnsecured != nil {
			return nil, utilerrors.NewFromKubernetesError(errGetUnsecured)
		}
		// Cluster is not ready yet, return original error
		return nil, utilerrors.NewFromKubernetesError(err)
	}
	return cluster, nil
}

func isStatus(err error, status int32) bool {
	var statusErr *apierrors.StatusError

	return errors.As(err, &statusErr) && status == statusErr.Status().Code
}
