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
	"errors"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
)

const (
	resourceNamePrefix         = "kubernetes-"
	kubernetesClusterTagPrefix = "kubernetes.io/cluster/"
	ownershipTagPrefix         = "owned-by.kubermatic.k8c.io/"

	regionAnnotationKey = "kubermatic.io/aws-region"

	cleanupFinalizer = "kubermatic.k8c.io/cleanup-aws"

	// The individual finalizers are deprecated and not used for newly reconciled
	// clusters, where the single cleanupFinalizer is enough.

	securityGroupCleanupFinalizer    = "kubermatic.k8c.io/cleanup-aws-security-group"
	instanceProfileCleanupFinalizer  = "kubermatic.k8c.io/cleanup-aws-instance-profile"
	controlPlaneRoleCleanupFinalizer = "kubermatic.k8c.io/cleanup-aws-control-plane-role"
	tagCleanupFinalizer              = "kubermatic.k8c.io/cleanup-aws-tags"

	authFailure = "AuthFailure"
)

type AmazonEC2 struct {
	dc                *kubermaticv1.DatacenterSpecAWS
	secretKeySelector provider.SecretKeySelectorValueFunc

	// clientSet is used during tests only
	clientSet *ClientSet
}

// NewCloudProvider returns a new AmazonEC2 provider.
func NewCloudProvider(dc *kubermaticv1.Datacenter, secretKeyGetter provider.SecretKeySelectorValueFunc) (*AmazonEC2, error) {
	if dc.Spec.AWS == nil {
		return nil, errors.New("datacenter is not an AWS datacenter")
	}

	return &AmazonEC2{
		dc:                dc.Spec.AWS,
		secretKeySelector: secretKeyGetter,
	}, nil
}

var _ provider.ReconcilingCloudProvider = &AmazonEC2{}

func (a *AmazonEC2) getClientSet(ctx context.Context, cloud kubermaticv1.CloudSpec) (*ClientSet, error) {
	if a.clientSet != nil {
		return a.clientSet, nil
	}

	accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, err := GetCredentialsForCluster(cloud, a.secretKeySelector)
	if err != nil {
		return nil, err
	}

	return GetClientSet(ctx, accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, a.dc.Region)
}

func (a *AmazonEC2) DefaultCloudSpec(ctx context.Context, spec *kubermaticv1.CloudSpec) error {
	return nil
}

// ValidateCloudSpec validates the fields that the user can override while creating
// a cluster. We only check those that must pre-exist in the AWS account
// (i.e. the security group and VPC), because the others (like route table)
// will be created if they do not yet exist / are not explicitly specified.
// TL;DR: This validation does not need to be extended to cover more than
// VPC and SG.
func (a *AmazonEC2) ValidateCloudSpec(ctx context.Context, spec kubermaticv1.CloudSpec) error {
	client, err := a.getClientSet(ctx, spec)
	if err != nil {
		return fmt.Errorf("failed to get API client: %w", err)
	}

	// Some settings require the vpc to be set
	if spec.AWS.SecurityGroupID != "" {
		if spec.AWS.VPCID == "" {
			return fmt.Errorf("VPC must be set when specifying a security group")
		}
	}

	if spec.AWS.VPCID != "" {
		vpc, err := getVPCByID(ctx, client.EC2, spec.AWS.VPCID)
		if err != nil {
			return err
		}

		if spec.AWS.SecurityGroupID != "" {
			if _, err = getSecurityGroupByID(ctx, client.EC2, vpc, spec.AWS.SecurityGroupID); err != nil {
				return err
			}
		}
	}

	return nil
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted.
func (a *AmazonEC2) ValidateCloudSpecUpdate(_ context.Context, oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	if oldSpec.AWS == nil || newSpec.AWS == nil {
		return errors.New("'aws' spec is empty")
	}

	if oldSpec.AWS.VPCID != "" && oldSpec.AWS.VPCID != newSpec.AWS.VPCID {
		return fmt.Errorf("updating AWS VPC ID is not supported (was %s, updated to %s)", oldSpec.AWS.VPCID, newSpec.AWS.VPCID)
	}

	if oldSpec.AWS.RouteTableID != "" && oldSpec.AWS.RouteTableID != newSpec.AWS.RouteTableID {
		return fmt.Errorf("updating AWS route table ID is not supported (was %s, updated to %s)", oldSpec.AWS.RouteTableID, newSpec.AWS.RouteTableID)
	}

	if oldSpec.AWS.SecurityGroupID != "" && oldSpec.AWS.SecurityGroupID != newSpec.AWS.SecurityGroupID {
		return fmt.Errorf("updating AWS security group ID is not supported (was %s, updated to %s)", oldSpec.AWS.SecurityGroupID, newSpec.AWS.SecurityGroupID)
	}

	if oldSpec.AWS.ControlPlaneRoleARN != "" && oldSpec.AWS.ControlPlaneRoleARN != newSpec.AWS.ControlPlaneRoleARN {
		return fmt.Errorf("updating AWS control plane ARN is not supported (was %s, updated to %s)", oldSpec.AWS.ControlPlaneRoleARN, newSpec.AWS.ControlPlaneRoleARN)
	}

	if oldSpec.AWS.InstanceProfileName != "" && oldSpec.AWS.InstanceProfileName != newSpec.AWS.InstanceProfileName {
		return fmt.Errorf("updating AWS instance profile name is not supported (was %s, updated to %s)", oldSpec.AWS.InstanceProfileName, newSpec.AWS.InstanceProfileName)
	}

	return nil
}

func (a *AmazonEC2) InitializeCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	// Initialization should only occur once.
	firstInitialization := !kuberneteshelper.HasFinalizer(cluster, cleanupFinalizer)

	return a.reconcileCluster(ctx, cluster, update, false, firstInitialization)
}

func (a *AmazonEC2) ReconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return a.reconcileCluster(ctx, cluster, update, true, true)
}

func (a *AmazonEC2) reconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, force bool, setTags bool) (*kubermaticv1.Cluster, error) {
	client, err := a.getClientSet(ctx, cluster.Spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get API client: %w", err)
	}

	cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		kuberneteshelper.AddFinalizer(cluster, cleanupFinalizer)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add finalizer: %w", err)
	}

	// update VPC ID
	if force || cluster.Spec.Cloud.AWS.VPCID == "" {
		cluster, err = reconcileVPC(ctx, client.EC2, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	// update route table ID
	if force || cluster.Spec.Cloud.AWS.RouteTableID == "" {
		cluster, err = reconcileRouteTable(ctx, client.EC2, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	// All machines will live in one dedicated security group.
	if force || cluster.Spec.Cloud.AWS.SecurityGroupID == "" {
		cluster, err = reconcileSecurityGroup(ctx, client.EC2, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	// We create a dedicated role for the control plane.
	if force || cluster.Spec.Cloud.AWS.ControlPlaneRoleARN == "" {
		cluster, err = reconcileControlPlaneRole(ctx, client.IAM, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	// instance profile and role for worker nodes
	if force || cluster.Spec.Cloud.AWS.InstanceProfileName == "" {
		cluster, err = reconcileWorkerInstanceProfile(ctx, client.IAM, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	// We put this as an annotation on the cluster to allow addons to read this
	// information.
	cluster, err = reconcileRegionAnnotation(ctx, cluster, update, a.dc.Region)
	if err != nil {
		return nil, err
	}

	// update resource ownership for older clusters
	cluster, err = backfillOwnershipTags(ctx, client, cluster, update)
	if err != nil {
		return nil, err
	}

	// tag all resources
	if setTags {
		cluster, err = reconcileClusterTags(ctx, client.EC2, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

func (a *AmazonEC2) CleanUpCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, updater provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	// prevent excessive requests to AWS when a cluster is re-reconciled often
	// during its deletion phase
	if !kuberneteshelper.HasFinalizer(cluster, cleanupFinalizer) {
		return cluster, nil
	}

	client, err := a.getClientSet(ctx, cluster.Spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get API client: %w", err)
	}

	// worker instance profile + role
	if err := cleanUpWorkerInstanceProfile(ctx, client.IAM, cluster); err != nil {
		return nil, fmt.Errorf("failed to clean up worker instance profile: %w", err)
	}

	// control plane role
	if err := cleanUpControlPlaneRole(ctx, client.IAM, cluster); err != nil {
		return nil, fmt.Errorf("failed to clean up control plane role: %w", err)
	}

	// security group
	if err := cleanUpSecurityGroup(ctx, client.EC2, cluster); err != nil {
		return nil, fmt.Errorf("failed to clean up security group: %w", err)
	}

	// No cleanup required for the route table itself.
	// No cleanup required for the VPC itself.

	// remove cluster tags
	if err := cleanUpTags(ctx, client.EC2, cluster); err != nil {
		return nil, fmt.Errorf("failed to clean up tags: %w", err)
	}

	return updater(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		kuberneteshelper.RemoveFinalizer(cluster, securityGroupCleanupFinalizer, controlPlaneRoleCleanupFinalizer, instanceProfileCleanupFinalizer, tagCleanupFinalizer, cleanupFinalizer)
	})
}
