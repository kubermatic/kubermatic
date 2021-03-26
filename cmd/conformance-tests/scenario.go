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

package main

import (
	"context"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

type testScenario interface {
	Name() string
	Cluster(secrets secrets) *apimodels.CreateClusterSpec
	NodeDeployments(ctx context.Context, num int, secrets secrets) ([]apimodels.NodeDeployment, error)
	OS() apimodels.OperatingSystemSpec
}

func supportsStorage(cluster *kubermaticv1.Cluster) bool {
	return cluster.Spec.Cloud.Openstack != nil ||
		cluster.Spec.Cloud.Azure != nil ||
		cluster.Spec.Cloud.AWS != nil ||
		cluster.Spec.Cloud.VSphere != nil ||
		cluster.Spec.Cloud.GCP != nil ||
		cluster.Spec.Cloud.Hetzner != nil
}

func supportsLBs(cluster *kubermaticv1.Cluster) bool {
	return cluster.Spec.Cloud.Azure != nil ||
		cluster.Spec.Cloud.AWS != nil ||
		cluster.Spec.Cloud.GCP != nil ||
		(cluster.Spec.Cloud.Hetzner != nil && cluster.Spec.Version.Minor() >= 18)
}
