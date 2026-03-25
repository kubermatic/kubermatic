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

func TestGetDefaultRouteTable(t *testing.T) {
	ctx := context.Background()
	cs := getTestClientSet(ctx, t)

	defaultVPC, err := getDefaultVPC(ctx, cs.EC2)
	if err != nil {
		t.Fatalf("getDefaultVPC should not have errored, but returned %v", err)
	}

	t.Run("valid-vpc", func(t *testing.T) {
		rt, err := getDefaultRouteTable(ctx, cs.EC2, *defaultVPC.VpcId)
		if err != nil {
			t.Fatalf("getDefaultRouteTable should not have errored, but returned %v", err)
		}

		if rt == nil {
			t.Fatal("getDefaultRouteTable should have found a default route table")
		}
	})

	t.Run("invalid-vpc", func(t *testing.T) {
		_, err := getDefaultRouteTable(ctx, cs.EC2, "does-not-exist")
		if err == nil {
			t.Fatalf("getDefaultRouteTable should have errored, but returned %v", err)
		}
	})
}

func TestReconcileRouteTable(t *testing.T) {
	ctx := context.Background()
	cs := getTestClientSet(ctx, t)

	defaultVPC, err := getDefaultVPC(ctx, cs.EC2)
	if err != nil {
		t.Fatalf("getDefaultVPC should not have errored, but returned %v", err)
	}

	defaultRT, err := getDefaultRouteTable(ctx, cs.EC2, *defaultVPC.VpcId)
	if err != nil {
		t.Fatalf("getDefaultRouteTable should not have errored, but returned %v", err)
	}

	defaultVPCID := *defaultVPC.VpcId
	defaultRouteTableID := *defaultRT.RouteTableId

	t.Run("everything-is-fine", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID:           defaultVPCID,
			RouteTableID:    defaultRouteTableID,
			AccessKeyID:     nope,
			SecretAccessKey: nope,
		})

		cluster, err = reconcileRouteTable(ctx, cs.EC2, cluster, testClusterUpdater(cluster))
		if err != nil {
			t.Fatalf("reconcileRouteTable should not have errored, but returned %v", err)
		}

		if cluster.Spec.Cloud.AWS.VPCID != defaultVPCID {
			t.Errorf("cloud spec should have retained VPC ID %q, but is now %q", defaultVPCID, cluster.Spec.Cloud.AWS.VPCID)
		}

		if cluster.Spec.Cloud.AWS.RouteTableID != defaultRouteTableID {
			t.Errorf("cloud spec should have retained route table ID %q, but is now %q", defaultRouteTableID, cluster.Spec.Cloud.AWS.RouteTableID)
		}
	})

	t.Run("no-route-table-yet", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			AccessKeyID:     nope,
			SecretAccessKey: nope,
			VPCID:           defaultVPCID,
		})

		cluster, err = reconcileRouteTable(ctx, cs.EC2, cluster, testClusterUpdater(cluster))
		if err != nil {
			t.Fatalf("reconcileRouteTable should not have errored, but returned %v", err)
		}

		if cluster.Spec.Cloud.AWS.VPCID != defaultVPCID {
			t.Errorf("cloud spec should have retained VPC ID %q, but is now %q", defaultVPCID, cluster.Spec.Cloud.AWS.VPCID)
		}

		if cluster.Spec.Cloud.AWS.RouteTableID != defaultRouteTableID {
			t.Errorf("cloud spec should have found route table ID %q, but is now %q", defaultRouteTableID, cluster.Spec.Cloud.AWS.RouteTableID)
		}
	})

	t.Run("invalid-route-table", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			AccessKeyID:     nope,
			SecretAccessKey: nope,
			VPCID:           defaultVPCID,
			RouteTableID:    "does-not-exist",
		})

		cluster, err = reconcileRouteTable(ctx, cs.EC2, cluster, testClusterUpdater(cluster))
		if err != nil {
			t.Fatalf("reconcileRouteTable should not have errored, but returned %v", err)
		}

		if cluster.Spec.Cloud.AWS.VPCID != defaultVPCID {
			t.Errorf("cloud spec should have retained VPC ID %q, but is now %q", defaultVPCID, cluster.Spec.Cloud.AWS.VPCID)
		}

		if cluster.Spec.Cloud.AWS.RouteTableID != defaultRouteTableID {
			t.Errorf("cloud spec should have fixedthe route table ID to %q, but is now %q", defaultRouteTableID, cluster.Spec.Cloud.AWS.RouteTableID)
		}
	})
}
