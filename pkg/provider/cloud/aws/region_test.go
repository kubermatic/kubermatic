//go:build integration

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

package aws

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

func TestReconcileRegionAnnotation(t *testing.T) {
	cluster := makeCluster(&kubermaticv1.AWSCloudSpec{})
	updater := testClusterUpdater(cluster)
	region := "test"

	// reconcile
	var err error
	cluster, err = reconcileRegionAnnotation(context.Background(), cluster, updater, region)
	if err != nil {
		t.Fatalf("reconcileRegionAnnotation should not have errored, but returned %v", err)
	}

	if cluster.Annotations[regionAnnotationKey] != region {
		t.Fatalf("region annotation should be %q, but is %q", region, cluster.Annotations[regionAnnotationKey])
	}
}

func TestReconcileRegionAnnotationFixingBadAnnotation(t *testing.T) {
	cluster := makeCluster(&kubermaticv1.AWSCloudSpec{})
	updater := testClusterUpdater(cluster)
	region := "test"

	// break the cluster
	badRegion := "not-" + region
	cluster.Annotations = map[string]string{regionAnnotationKey: badRegion}

	// fix it
	var err error
	cluster, err = reconcileRegionAnnotation(context.Background(), cluster, updater, region)
	if err != nil {
		t.Fatalf("reconcileRegionAnnotation should not have errored, but returned %v", err)
	}

	if cluster.Annotations[regionAnnotationKey] != region {
		t.Fatalf("region annotation should be %q, but is %q", region, badRegion)
	}
}
