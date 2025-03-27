//go:build !ee

/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package metering

import (
	"context"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileMeteringResources reconciles the metering related resources.
func ReconcileMeteringResources(_ context.Context, _ ctrlruntimeclient.Client, _ *runtime.Scheme, _ *kubermaticv1.KubermaticConfiguration, _ *kubermaticv1.Seed) error {
	return nil
}

// CronJobReconciler returns the func to create/update the metering report cronjob. Available only for ee.
func CronJobReconciler(_ string, _ kubermaticv1.MeteringReportConfiguration, _ string, _ registry.ImageRewriter, _ *kubermaticv1.Seed) reconciling.NamedCronJobReconcilerFactory {
	return nil
}

// MeteringPrometheusReconciler returns the func to create/update the metering prometheus statefulset. Available only for ee.
func MeteringPrometheusReconciler(_ registry.ImageRewriter, _ *kubermaticv1.Seed) reconciling.NamedStatefulSetReconcilerFactory {
	return nil
}
