//go:build !ee

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

package validation

import (
	"context"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"

	"k8s.io/apimachinery/pkg/labels"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func validateQuota(_ context.Context, _ *zap.SugaredLogger, _ ctrlruntimeclient.Client, _ *clusterv1alpha1.Machine,
	_ *certificates.CABundle, _ *kubermaticv1.ResourceQuota) error {
	return nil
}

// Resource Quotas are an EE feature
func getResourceQuota(_ context.Context, _ ctrlruntimeclient.Client, _ labels.Selector) (*kubermaticv1.ResourceQuota, error) {
	return nil, nil
}
