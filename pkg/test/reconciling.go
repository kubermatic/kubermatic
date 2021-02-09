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

package test

import (
	"context"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
)

// NewTemplateData returns a new TemplateData to be used for testing purposes.
func NewTemplateData(ctx context.Context, cli client.Client, cluster *kubermaticv1.Cluster) *resources.TemplateData {
	return resources.NewTemplateData(
		ctx,
		cli,
		cluster,
		&kubermaticv1.Datacenter{},
		&kubermaticv1.Seed{
			ObjectMeta: metav1.ObjectMeta{Name: "testdc"},
			Spec: kubermaticv1.SeedSpec{
				ProxySettings: &kubermaticv1.ProxySettings{
					HTTPProxy: kubermaticv1.NewProxyValue("http://my-corp"),
				},
			},
		},
		"",
		"",
		"192.0.2.0/24",
		resource.MustParse("5Gi"),
		"kubermatic_io_monitoring",
		"",
		false,
		false,
		"",
		"test",
		"https://dev.kubermatic.io/dex",
		"kubermaticIssuer",
		true,
		"quay.io/kubermatic/kubermatic",
		"quay.io/kubermatic/etcd-launcher",
		"quay.io/kubermatic/kubeletdnat-controller",
		false,
		kubermatic.NewFakeVersions(),
	)
}
