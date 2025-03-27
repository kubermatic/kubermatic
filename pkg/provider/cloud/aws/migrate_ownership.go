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
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	iam "github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/utils/ptr"
)

// backfillOwnershipTags migrates existing clusters to the new reconciling.
// Back when we only did a one-time initialization, resources had no
// ownership tag. Now with the new reconciling implementation, we need
// this tag to properly clean up and not create leftover resources.
// This is critical for cases where users configured their own security
// group or profile names, and removing them could potentially disrupt
// other clusters (though of course, we reconcile those and would fix
// it, but then KKP owns the resources which previously were owned by
// the user).
//
// To deduce ownership, we look at the finalizers, which in the old
// implementation were added for each resource, in order to minimize
// the potential API calls when cleaning up. This means that if for
// example the security group finalizer exists, KKP owns it.
//
// This function will update the EC2/IAM resources and remove the now
// unused finalizer afterwards. This should happen after the other
// reconciling functions have been called, so that this function can
// assume this like the security group ID are set and valid.
func backfillOwnershipTags(ctx context.Context, cs *ClientSet, cluster *kubermaticv1.Cluster, updater provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	ec2Tag := ec2OwnershipTag(cluster.Name)
	iamTag := iamOwnershipTag(cluster.Name)

	// the tag finalizer is of no use at all
	cluster, err := updater(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		kuberneteshelper.RemoveFinalizer(cluster, tagCleanupFinalizer)
	})
	if err != nil {
		return cluster, fmt.Errorf("failed to remove finalizer %q: %w", controlPlaneRoleCleanupFinalizer, err)
	}

	// security group
	if kuberneteshelper.HasFinalizer(cluster, securityGroupCleanupFinalizer) {
		vpc, err := getVPCByID(ctx, cs.EC2, cluster.Spec.Cloud.AWS.VPCID)
		if err != nil {
			return cluster, fmt.Errorf("failed to get VPC: %w", err)
		}

		group, err := getSecurityGroupByID(ctx, cs.EC2, vpc, cluster.Spec.Cloud.AWS.SecurityGroupID)
		if err != nil {
			return cluster, fmt.Errorf("failed to get security group: %w", err)
		}

		if !hasEC2Tag(ec2Tag, group.Tags) {
			_, err = cs.EC2.CreateTags(ctx, &ec2.CreateTagsInput{
				Resources: []string{ptr.Deref(group.GroupId, "")},
				Tags:      []ec2types.Tag{ec2Tag},
			})
			if err != nil {
				return cluster, fmt.Errorf("failed to tag security group: %w", err)
			}
		}

		cluster, err = updater(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, securityGroupCleanupFinalizer)
		})
		if err != nil {
			return cluster, fmt.Errorf("failed to remove finalizer %q: %w", securityGroupCleanupFinalizer, err)
		}
	}

	// instance profile
	if kuberneteshelper.HasFinalizer(cluster, instanceProfileCleanupFinalizer) {
		profile, err := getInstanceProfile(ctx, cs.IAM, cluster.Spec.Cloud.AWS.InstanceProfileName)
		if err != nil {
			return cluster, fmt.Errorf("failed to get instance profile: %w", err)
		}

		if !hasIAMTag(iamTag, profile.Tags) {
			_, err = cs.IAM.TagInstanceProfile(ctx, &iam.TagInstanceProfileInput{
				InstanceProfileName: profile.InstanceProfileName,
				Tags:                []iamtypes.Tag{iamTag},
			})
			if err != nil {
				return cluster, fmt.Errorf("failed to tag instance profile: %w", err)
			}
		}

		cluster, err = updater(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, instanceProfileCleanupFinalizer)
		})
		if err != nil {
			return cluster, fmt.Errorf("failed to remove finalizer %q: %w", instanceProfileCleanupFinalizer, err)
		}
	}

	// control plane role
	if kuberneteshelper.HasFinalizer(cluster, controlPlaneRoleCleanupFinalizer) {
		role, err := getRole(ctx, cs.IAM, cluster.Spec.Cloud.AWS.ControlPlaneRoleARN)
		if err != nil {
			return cluster, fmt.Errorf("failed to get control plane role: %w", err)
		}

		if !hasIAMTag(iamTag, role.Tags) {
			_, err = cs.IAM.TagRole(ctx, &iam.TagRoleInput{
				RoleName: role.RoleName,
				Tags:     []iamtypes.Tag{iamTag},
			})
			if err != nil {
				return cluster, fmt.Errorf("failed to tag control plane role: %w", err)
			}
		}

		cluster, err = updater(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, controlPlaneRoleCleanupFinalizer)
		})
		if err != nil {
			return cluster, fmt.Errorf("failed to remove finalizer %q: %w", controlPlaneRoleCleanupFinalizer, err)
		}
	}

	return cluster, err
}
