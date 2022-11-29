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

package csi

import (
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/csi/kubevirt"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
)

// ServiceAccountsCreators returns the function to create and update the service accounts needed for CSI.
func ServiceAccountCreators(cluster *kubermaticv1.Cluster) []reconciling.NamedServiceAccountReconcilerFactory {
	creatorGetters := []reconciling.NamedServiceAccountReconcilerFactory{}

	switch {
	case cluster.Spec.Cloud.Kubevirt != nil:
		creatorGetters = append(creatorGetters, kubevirt.ServiceAccountsCreators(cluster)...)
	}

	return creatorGetters
}

// ClusterRolesCreators returns the function to create and update the clusterroles needed for CSI.
func ClusterRolesCreators(c *kubermaticv1.Cluster) []reconciling.NamedClusterRoleReconcilerFactory {
	creatorGetters := []reconciling.NamedClusterRoleReconcilerFactory{}

	switch {
	case c.Spec.Cloud.Kubevirt != nil:
		creatorGetters = kubevirt.ClusterRolesCreators()
	}

	return creatorGetters
}

// RoleBindingsCreators returns the function to create and update the rolebindings needed for CSI.
func RoleBindingsCreators(c *kubermaticv1.Cluster) []reconciling.NamedRoleBindingReconcilerFactory {
	creatorGetters := []reconciling.NamedRoleBindingReconcilerFactory{}

	switch {
	case c.Spec.Cloud.Kubevirt != nil:
		creatorGetters = kubevirt.RoleBindingsCreators(c)
	}

	return creatorGetters
}
