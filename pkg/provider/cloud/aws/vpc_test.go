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

func TestGetDefaultVPC(t *testing.T) {
	ctx := context.Background()
	cs := getTestClientSet(ctx, t)

	result, err := getDefaultVPC(context.Background(), cs.EC2)
	if err != nil {
		t.Fatalf("getDefaultVPC should not have errored, but returned %v", err)
	}

	if result == nil {
		t.Fatal("getDefaultVPC should have found a default VPC")
	}
}

func TestGetVPCByID(t *testing.T) {
	ctx := context.Background()
	cs := getTestClientSet(ctx, t)

	t.Run("default-vpc", func(t *testing.T) {
		defaultVPC, err := getDefaultVPC(ctx, cs.EC2)
		if err != nil {
			t.Fatalf("getDefaultVPC should not have errored, but returned %v", err)
		}

		vpcID := *defaultVPC.VpcId

		other, err := getVPCByID(ctx, cs.EC2, vpcID)
		if err != nil {
			t.Fatalf("getVPCByID should not have errored, but returned %v", err)
		}

		if *other.VpcId != vpcID {
			t.Fatalf("getVPCByID should have returned VPC %q, but returned %q instead.", vpcID, *other.VpcId)
		}
	})

	t.Run("nonexisting-vpc", func(t *testing.T) {
		if _, err := getVPCByID(ctx, cs.EC2, "does-not-exist"); err == nil {
			t.Fatalf("getVPCByID should have errored, but returned %v", err)
		}
	})
}

func TestReconcileVPC(t *testing.T) {
	ctx := context.Background()
	cs := getTestClientSet(ctx, t)

	defaultVPC, err := getDefaultVPC(ctx, cs.EC2)
	if err != nil {
		t.Fatalf("getDefaultVPC should not have errored, but returned %v", err)
	}

	defaultVPCID := *defaultVPC.VpcId

	t.Run("cluster-already-reconciled", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID: defaultVPCID,
		})

		cluster, err = reconcileVPC(ctx, cs.EC2, cluster, testClusterUpdater(cluster))
		if err != nil {
			t.Fatalf("reconcileVPC should not have errored, but returned %v", err)
		}

		if cluster.Spec.Cloud.AWS.VPCID != defaultVPCID {
			t.Fatalf("cloud spec should have retained VPC ID %q, but is now %q", defaultVPCID, cluster.Spec.Cloud.AWS.VPCID)
		}
	})

	t.Run("no-vpc-set-in-cluster", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{})

		cluster, err = reconcileVPC(ctx, cs.EC2, cluster, testClusterUpdater(cluster))
		if err != nil {
			t.Fatalf("reconcileVPC should not have errored, but returned %v", err)
		}

		if cluster.Spec.Cloud.AWS.VPCID != defaultVPCID {
			t.Fatalf("cloud spec should have found default VPC with ID %q, but is now %q", defaultVPCID, cluster.Spec.Cloud.AWS.VPCID)
		}
	})

	t.Run("invalid-vpc-configured", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID: "does-not-exist",
		})

		if _, err = reconcileVPC(ctx, cs.EC2, cluster, testClusterUpdater(cluster)); err == nil {
			t.Fatalf("reconcileVPC should have errored, but returned %v", err)
		}
	})
}
