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

	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/ptr"
)

func TestEnsureInstanceProfile(t *testing.T) {
	ctx := context.Background()
	cs := getTestClientSet(ctx, t)

	defaultVPC, err := getDefaultVPC(ctx, cs.EC2)
	if err != nil {
		t.Fatalf("getDefaultVPC should not have errored, but returned %v", err)
	}
	defaultVPCID := *defaultVPC.VpcId

	t.Run("create-new-profile", func(t *testing.T) {
		profileName := "test-" + rand.String(10)
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID: defaultVPCID,
		})

		profile, err := ensureInstanceProfile(ctx, cs.IAM, cluster, profileName)
		if err != nil {
			t.Fatalf("ensureInstanceProfile should not have errored, but returned %v", err)
		}

		if *profile.InstanceProfileName != profileName {
			t.Errorf("expected profile name %q, but found %q", profileName, *profile.InstanceProfileName)
		}

		if !hasIAMTag(iamOwnershipTag(cluster.Name), profile.Tags) {
			t.Errorf("expected profile to have ownership tag, but does not")
		}

		// doing it again should not cause any harm
		profile, err = ensureInstanceProfile(ctx, cs.IAM, cluster, profileName)
		if err != nil {
			t.Fatalf("ensureInstanceProfile (2) should not have errored, but returned %v", err)
		}

		// we should still own the profile, as it's the same one from earlier
		if !hasIAMTag(iamOwnershipTag(cluster.Name), profile.Tags) {
			t.Errorf("expected profile to have ownership tag, but does not")
		}
	})

	t.Run("adopt-existing-profile", func(t *testing.T) {
		// create a profile
		profileName := "test-" + rand.String(10)

		// no ownership tag here, we want to simulate a normal, pre-existing profile
		createProfileInput := &iam.CreateInstanceProfileInput{
			InstanceProfileName: ptr.To(profileName),
		}

		if _, err := cs.IAM.CreateInstanceProfile(ctx, createProfileInput); err != nil {
			t.Fatalf("CreateInstanceProfile should not have errored, but returned %v", err)
		}

		// tell the cluster to use the existing profile
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID:               defaultVPCID,
			InstanceProfileName: profileName,
		})

		profile, err := ensureInstanceProfile(ctx, cs.IAM, cluster, profileName)
		if err != nil {
			t.Fatalf("ensureInstanceProfile should not have errored, but returned %v", err)
		}

		// we should now NOT own the profile
		if hasIAMTag(iamOwnershipTag(cluster.Name), profile.Tags) {
			t.Errorf("expected profile to not have ownership tag, but it does")
		}
	})
}

func profileHasRole(profile *iamtypes.InstanceProfile, roleName string) bool {
	for _, role := range profile.Roles {
		if *role.RoleName == roleName {
			return true
		}
	}

	return false
}

func TestReconcileWorkerInstanceProfile(t *testing.T) {
	ctx := context.Background()
	cs := getTestClientSet(ctx, t)

	defaultVPC, err := getDefaultVPC(ctx, cs.EC2)
	if err != nil {
		t.Fatalf("getDefaultVPC should not have errored, but returned %v", err)
	}
	defaultVPCID := *defaultVPC.VpcId

	t.Run("create-new-profile-and-role", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID: defaultVPCID,
		})

		cluster, err = reconcileWorkerInstanceProfile(ctx, cs.IAM, cluster, testClusterUpdater(cluster))
		if err != nil {
			t.Fatalf("reconcileWorkerInstanceProfile should not have errored, but returned %v", err)
		}

		if cluster.Spec.Cloud.AWS.InstanceProfileName == "" {
			t.Error("Cluster spec should have an instance profile name set, but it's empty")
		}

		profile, err := getInstanceProfile(ctx, cs.IAM, cluster.Spec.Cloud.AWS.InstanceProfileName)
		if err != nil {
			t.Fatalf("getInstanceProfile should not have errored, but returned %v", err)
		}

		if !hasIAMTag(iamOwnershipTag(cluster.Name), profile.Tags) {
			t.Errorf("expected profile to have ownership tag, but does not")
		}

		if !profileHasRole(profile, workerRoleName(cluster.Name)) {
			t.Errorf("expected profile to have worker role, but does not")
		}
	})

	t.Run("keep-name-when-fixing-missing-profile", func(t *testing.T) {
		profileName := "test-" + rand.String(10)
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID:               defaultVPCID,
			InstanceProfileName: profileName,
		})

		// this will create a new profile that is owned by us
		cluster, err = reconcileWorkerInstanceProfile(ctx, cs.IAM, cluster, testClusterUpdater(cluster))
		if err != nil {
			t.Fatalf("reconcileWorkerInstanceProfile should not have errored, but returned %v", err)
		}

		if cluster.Spec.Cloud.AWS.InstanceProfileName != profileName {
			t.Errorf("Cluster spec should have retained profile name %q, but now is %q", profileName, cluster.Spec.Cloud.AWS.InstanceProfileName)
		}

		profile, err := getInstanceProfile(ctx, cs.IAM, cluster.Spec.Cloud.AWS.InstanceProfileName)
		if err != nil {
			t.Fatalf("getInstanceProfile should not have errored, but returned %v", err)
		}

		if !hasIAMTag(iamOwnershipTag(cluster.Name), profile.Tags) {
			t.Errorf("expected profile to have ownership tag, but does not")
		}

		if !profileHasRole(profile, workerRoleName(cluster.Name)) {
			t.Errorf("expected profile to have worker role, but does not")
		}
	})

	t.Run("use-foreign-profile", func(t *testing.T) {
		// create a profile
		profileName := "test-" + rand.String(10)

		// no ownership tag here, we want to simulate a normal, pre-existing profile
		createProfileInput := &iam.CreateInstanceProfileInput{
			InstanceProfileName: ptr.To(profileName),
		}

		if _, err := cs.IAM.CreateInstanceProfile(ctx, createProfileInput); err != nil {
			t.Fatalf("CreateInstanceProfile should not have errored, but returned %v", err)
		}

		// prepare a cluster that uses the foreign profile
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID:               defaultVPCID,
			InstanceProfileName: profileName,
		})

		// this should create neither a profile nor a role, we rely entirely on the pre-existing stuff,
		// no matter how broken it might be (it's the user's responsibility if they make us use their
		// profile)
		cluster, err = reconcileWorkerInstanceProfile(ctx, cs.IAM, cluster, testClusterUpdater(cluster))
		if err != nil {
			t.Fatalf("reconcileWorkerInstanceProfile should not have errored, but returned %v", err)
		}

		// this should not have changed
		if cluster.Spec.Cloud.AWS.InstanceProfileName != profileName {
			t.Errorf("Cluster spec should have retained profile name %q, but now is %q", profileName, cluster.Spec.Cloud.AWS.InstanceProfileName)
		}

		profile, err := getInstanceProfile(ctx, cs.IAM, cluster.Spec.Cloud.AWS.InstanceProfileName)
		if err != nil {
			t.Fatalf("getInstanceProfile should not have errored, but returned %v", err)
		}

		if hasIAMTag(iamOwnershipTag(cluster.Name), profile.Tags) {
			t.Errorf("expected profile to not have ownership tag, but it does")
		}

		if profileHasRole(profile, workerRoleName(cluster.Name)) {
			t.Errorf("when using pre-existing profiles, we should not have created our own role")
		}
	})
}

func TestCleanUpWorkerInstanceProfile(t *testing.T) {
	ctx := context.Background()
	cs := getTestClientSet(ctx, t)

	defaultVPC, err := getDefaultVPC(ctx, cs.EC2)
	if err != nil {
		t.Fatalf("getDefaultVPC should not have errored, but returned %v", err)
	}
	defaultVPCID := *defaultVPC.VpcId

	t.Run("vanilla-case-where-we-own-everything", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID: defaultVPCID,
		})

		cluster, err = reconcileWorkerInstanceProfile(ctx, cs.IAM, cluster, testClusterUpdater(cluster))
		if err != nil {
			t.Fatalf("reconcileWorkerInstanceProfile should not have errored, but returned %v", err)
		}

		profileName := cluster.Spec.Cloud.AWS.InstanceProfileName

		if err = cleanUpWorkerInstanceProfile(ctx, cs.IAM, cluster); err != nil {
			t.Fatalf("cleanUpWorkerInstanceProfile should not have errored, but returned %v", err)
		}

		// make sure the profile is gone
		if _, err := getInstanceProfile(ctx, cs.IAM, profileName); err == nil {
			t.Fatal("getInstanceProfile should not have been able to find the profile, but it did")
		}

		// make sure the role is also gone
		if _, err := getRole(ctx, cs.IAM, workerRoleName(cluster.Name)); err == nil {
			t.Fatal("getRole should not have been able to find the worker role, but it did")
		}
	})

	t.Run("vanilla-case-but-we-lost-the-profile-name", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID: defaultVPCID,
		})

		cluster, err = reconcileWorkerInstanceProfile(ctx, cs.IAM, cluster, testClusterUpdater(cluster))
		if err != nil {
			t.Fatalf("reconcileWorkerInstanceProfile should not have errored, but returned %v", err)
		}

		profileName := cluster.Spec.Cloud.AWS.InstanceProfileName

		// the big difference to the vanilla-case testcase: we forget the profile name
		cluster.Spec.Cloud.AWS.InstanceProfileName = ""

		if err = cleanUpWorkerInstanceProfile(ctx, cs.IAM, cluster); err != nil {
			t.Fatalf("cleanUpWorkerInstanceProfile should not have errored, but returned %v", err)
		}

		// make sure the profile is gone
		if _, err := getInstanceProfile(ctx, cs.IAM, profileName); err == nil {
			t.Fatal("getInstanceProfile should not have been able to find the profile, but it did")
		}

		// make sure the role is also gone
		if _, err := getRole(ctx, cs.IAM, workerRoleName(cluster.Name)); err == nil {
			t.Fatal("getRole should not have been able to find the worker role, but it did")
		}
	})

	t.Run("everything-is-gone-already", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID: defaultVPCID,
		})

		if err = cleanUpWorkerInstanceProfile(ctx, cs.IAM, cluster); err != nil {
			t.Fatalf("cleanUpWorkerInstanceProfile should not have errored, but returned %v", err)
		}
	})

	t.Run("keep-foreign-profile-alive", func(t *testing.T) {
		// create a profile
		profileName := "test-" + rand.String(10)

		// no ownership tag here, we want to simulate a normal, pre-existing profile
		createProfileInput := &iam.CreateInstanceProfileInput{
			InstanceProfileName: ptr.To(profileName),
		}

		if _, err := cs.IAM.CreateInstanceProfile(ctx, createProfileInput); err != nil {
			t.Fatalf("CreateInstanceProfile should not have errored, but returned %v", err)
		}

		// prepare a cluster that uses the foreign profile
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID:               defaultVPCID,
			InstanceProfileName: profileName,
		})

		// clean it up, this should do nothing
		if err = cleanUpWorkerInstanceProfile(ctx, cs.IAM, cluster); err != nil {
			t.Fatalf("cleanUpWorkerInstanceProfile should not have errored, but returned %v", err)
		}

		// make sure the profile still exists
		if _, err := getInstanceProfile(ctx, cs.IAM, profileName); err != nil {
			t.Fatal("getInstanceProfile should have been able to find the foreign profile, but it is gone")
		}
	})
}
