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
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/iam"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog"
	"sigs.k8s.io/aws-iam-authenticator/pkg/token"
)

const (
	resourceNamePrefix = "kubernetes-"

	regionAnnotationKey = "kubermatic.io/aws-region"

	securityGroupCleanupFinalizer    = "kubermatic.io/cleanup-aws-security-group"
	instanceProfileCleanupFinalizer  = "kubermatic.io/cleanup-aws-instance-profile"
	controlPlaneRoleCleanupFinalizer = "kubermatic.io/cleanup-aws-control-plane-role"
	tagCleanupFinalizer              = "kubermatic.io/cleanup-aws-tags"

	tagNameKubernetesClusterPrefix = "kubernetes.io/cluster/"
	ownershipTagPrefix             = "owned-by.kubermatic.k8c.io/"

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

func (a *AmazonEC2) getClientSet(cloud kubermaticv1.CloudSpec) (*ClientSet, error) {
	if a.clientSet != nil {
		return a.clientSet, nil
	}

	accessKeyID, secretAccessKey, err := GetCredentialsForCluster(cloud, a.secretKeySelector)
	if err != nil {
		return nil, err
	}

	return GetClientSet(accessKeyID, secretAccessKey, a.dc.Region)
}

func (a *AmazonEC2) DefaultCloudSpec(spec *kubermaticv1.CloudSpec) error {
	return nil
}

// ValidateCloudSpec validates the fields that the user can override while creating
// a cluster. We only check those that must pre-exist in the AWS account
// (i.e. the security group and VPC), because the others (like route table)
// will be created if they do not yet exist / are not explicitly specified.
// TL;DR: This validation does not need to be extended to cover more than
// VPC and SG.
func (a *AmazonEC2) ValidateCloudSpec(spec kubermaticv1.CloudSpec) error {
	client, err := a.getClientSet(spec)
	if err != nil {
		return fmt.Errorf("failed to get API client: %v", err)
	}

	// Some settings require the vpc to be set
	if spec.AWS.SecurityGroupID != "" {
		if spec.AWS.VPCID == "" {
			return fmt.Errorf("VPC must be set when specifying a security group")
		}
	}

	if spec.AWS.VPCID != "" {
		vpc, err := getVPCByID(client.EC2, spec.AWS.VPCID)
		if err != nil {
			return err
		}

		if spec.AWS.SecurityGroupID != "" {
			if _, err = getSecurityGroupByID(client.EC2, vpc, spec.AWS.SecurityGroupID); err != nil {
				return err
			}
		}
	}

	return nil
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted
func (a *AmazonEC2) ValidateCloudSpecUpdate(oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}

// MigrateToMultiAZ migrates an AWS cluster from the old AZ-hardcoded spec to multi-AZ spec
func (a *AmazonEC2) MigrateToMultiAZ(cluster *kubermaticv1.Cluster, clusterUpdater provider.ClusterUpdater) error {
	// If not even the role name is set, then the cluster is not fully
	// initialized and we don't need to worry about this migration just yet.
	if cluster.Spec.Cloud.AWS.RoleName == "" {
		return nil
	}

	if cluster.Spec.Cloud.AWS.ControlPlaneRoleARN == "" {
		client, err := a.getClientSet(cluster.Spec.Cloud)
		if err != nil {
			return fmt.Errorf("failed to get API client: %v", err)
		}

		paramsRoleGet := &iam.GetRoleInput{RoleName: aws.String(cluster.Spec.Cloud.AWS.RoleName)}
		getRoleOut, err := client.IAM.GetRole(paramsRoleGet)
		if err != nil {
			return fmt.Errorf("failed to get already existing aws IAM role %s: %v", cluster.Spec.Cloud.AWS.RoleName, err)
		}

		newCluster, err := clusterUpdater(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.AWS.ControlPlaneRoleARN = *getRoleOut.Role.Arn
		})
		if err != nil {
			return err
		}
		*cluster = *newCluster
	}

	return nil
}

// AddICMPRulesIfRequired will create security rules that allow ICMP traffic if these do not yet exist.
// It is a part of a migration for older clusers (migrationRevision < 1) that didn't have these rules.
func (a *AmazonEC2) AddICMPRulesIfRequired(cluster *kubermaticv1.Cluster) error {
	if cluster.Spec.Cloud.AWS.SecurityGroupID == "" {
		klog.Infof("Not adding ICMP allow rules for cluster %q as it has no securityGroupID set",
			cluster.Name)
		return nil
	}

	client, err := a.getClientSet(cluster.Spec.Cloud)
	if err != nil {
		return fmt.Errorf("failed to get API client: %v", err)
	}
	out, err := client.EC2.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupIds: aws.StringSlice([]string{cluster.Spec.Cloud.AWS.SecurityGroupID}),
	})
	if err != nil {
		return fmt.Errorf("failed to get security group %q: %v", cluster.Spec.Cloud.AWS.SecurityGroupID, err)
	}

	// Should never happen
	if len(out.SecurityGroups) > 1 {
		return fmt.Errorf("got more than one(%d) security group for id %q",
			(len(out.SecurityGroups)), cluster.Spec.Cloud.AWS.SecurityGroupID)
	}
	if len(out.SecurityGroups) == 0 {
		return fmt.Errorf("did not find a security group for id %q",
			cluster.Spec.Cloud.AWS.SecurityGroupID)
	}

	var hasIPV4ICMPRule, hasIPV6ICMPRule bool
	for _, rule := range out.SecurityGroups[0].IpPermissions {
		if rule.FromPort != nil && *rule.FromPort == -1 && rule.ToPort != nil && *rule.ToPort == -1 {

			if *rule.IpProtocol == "icmp" && len(rule.IpRanges) == 1 && *rule.IpRanges[0].CidrIp == "0.0.0.0/0" {
				hasIPV4ICMPRule = true
			}
			if *rule.IpProtocol == "icmpv6" && len(rule.Ipv6Ranges) == 1 && *rule.Ipv6Ranges[0].CidrIpv6 == "::/0" {
				hasIPV6ICMPRule = true
			}
		}
	}

	var secGroupRules []*ec2.IpPermission
	if !hasIPV4ICMPRule {
		klog.Infof("Adding allow rule for icmp to cluster %q", cluster.Name)
		secGroupRules = append(secGroupRules,
			(&ec2.IpPermission{}).
				SetIpProtocol("icmp").
				SetFromPort(-1).
				SetToPort(-1).
				SetIpRanges([]*ec2.IpRange{
					{CidrIp: aws.String("0.0.0.0/0")},
				}))
	}
	if !hasIPV6ICMPRule {
		klog.Infof("Adding allow rule for icmpv6 to cluster %q", cluster.Name)
		secGroupRules = append(secGroupRules,
			(&ec2.IpPermission{}).
				SetIpProtocol("icmpv6").
				SetFromPort(-1).
				SetToPort(-1).
				SetIpv6Ranges([]*ec2.Ipv6Range{
					{CidrIpv6: aws.String("::/0")},
				}))
	}

	if len(secGroupRules) > 0 {
		_, err = client.EC2.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
			GroupId:       aws.String(cluster.Spec.Cloud.AWS.SecurityGroupID),
			IpPermissions: secGroupRules,
		})
		if err != nil {
			return fmt.Errorf("failed to add ICMP rules to security group %q: %v",
				cluster.Spec.Cloud.AWS.SecurityGroupID, err)
		}
	}

	return nil
}

func (a *AmazonEC2) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, reconcile bool) (*kubermaticv1.Cluster, error) {
	client, err := a.getClientSet(cluster.Spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get API client: %v", err)
	}

	// Initialization should only occur once. The regular reconciling, where we
	// check if all the resources still exist, happens in ReconcileCluster().

	if cluster.Spec.Cloud.AWS.VPCID == "" {
		cluster, err = reconcileVPC(client.EC2, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.AWS.RouteTableID == "" {
		cluster, err = reconcileRouteTable(client.EC2, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.AWS.SecurityGroupID == "" {
		cluster, err = reconcileSecurityGroup(client.EC2, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.AWS.ControlPlaneRoleARN == "" {
		cluster, err = reconcileControlPlaneRole(client.IAM, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.AWS.InstanceProfileName == "" {
		cluster, err = reconcileWorkerInstanceProfile(client.IAM, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	if !kuberneteshelper.HasFinalizer(cluster, tagCleanupFinalizer) {
		cluster, err = reconcileClusterTags(client.EC2, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	cluster, err = reconcileRegionAnnotation(cluster, update, a.dc.Region)
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

func (a *AmazonEC2) ReconcileCluster(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	client, err := a.getClientSet(cluster.Spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get API client: %v", err)
	}

	// update VPC ID
	cluster, err = reconcileVPC(client.EC2, cluster, update)
	if err != nil {
		return nil, err
	}

	// update route table ID
	cluster, err = reconcileRouteTable(client.EC2, cluster, update)
	if err != nil {
		return nil, err
	}

	// All machines will live in one dedicated security group.
	cluster, err = reconcileSecurityGroup(client.EC2, cluster, update)
	if err != nil {
		return nil, err
	}

	// We create a dedicated role for the control plane.
	cluster, err = reconcileControlPlaneRole(client.IAM, cluster, update)
	if err != nil {
		return nil, err
	}

	// instance profile and role for worker nodes
	cluster, err = reconcileWorkerInstanceProfile(client.IAM, cluster, update)
	if err != nil {
		return nil, err
	}

	// tag all resources
	cluster, err = reconcileClusterTags(client.EC2, cluster, update)
	if err != nil {
		return nil, err
	}

	// We put this as an annotation on the cluster to allow addons to read this
	// information.
	cluster, err = reconcileRegionAnnotation(cluster, update, a.dc.Region)
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

func (a *AmazonEC2) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, updater provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	client, err := a.getClientSet(cluster.Spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get API client: %w", err)
	}

	// worker instance profile + role
	if err := cleanUpWorkerInstanceProfile(client.IAM, cluster); err != nil {
		return nil, fmt.Errorf("failed to clean up worker instance profile: %w", err)
	}
	cluster, err = updater(cluster.Name, func(cluster *kubermaticv1.Cluster) {
		kuberneteshelper.RemoveFinalizer(cluster, instanceProfileCleanupFinalizer)
	})
	if err != nil {
		return nil, err
	}

	// control plane role
	if err := cleanUpControlPlaneRole(client.IAM, cluster); err != nil {
		return nil, fmt.Errorf("failed to clean up control plane role: %w", err)
	}
	cluster, err = updater(cluster.Name, func(cluster *kubermaticv1.Cluster) {
		kuberneteshelper.RemoveFinalizer(cluster, controlPlaneRoleCleanupFinalizer)
	})
	if err != nil {
		return nil, err
	}

	// security group
	if err := cleanUpSecurityGroup(client.EC2, cluster); err != nil {
		return nil, fmt.Errorf("failed to clean up security group: %w", err)
	}
	cluster, err = updater(cluster.Name, func(cluster *kubermaticv1.Cluster) {
		kuberneteshelper.RemoveFinalizer(cluster, securityGroupCleanupFinalizer)
	})
	if err != nil {
		return nil, err
	}

	// No cleanup required for the route table itself.
	// No cleanup required for the VPC itself.

	// tags
	if err := cleanUpTags(client.EC2, cluster); err != nil {
		return nil, fmt.Errorf("failed to clean up tags: %w", err)
	}
	cluster, err = updater(cluster.Name, func(cluster *kubermaticv1.Cluster) {
		kuberneteshelper.RemoveFinalizer(cluster, tagCleanupFinalizer)
	})
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

func GetEKSClusterConfig(ctx context.Context, accessKeyID, secretAccessKey, clusterName, region string) (*api.Config, error) {
	sess, err := getAWSSession(accessKeyID, secretAccessKey, region)
	if err != nil {
		return nil, err
	}
	eksSvc := eks.New(sess)

	clusterInput := &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	}
	clusterOutput, err := eksSvc.DescribeCluster(clusterInput)
	if err != nil {
		return nil, fmt.Errorf("error calling DescribeCluster: %v", err)
	}

	cluster := clusterOutput.Cluster
	eksclusterName := cluster.Name

	config := api.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters:   map[string]*api.Cluster{},  // Clusters is a map of referencable names to cluster configs
		AuthInfos:  map[string]*api.AuthInfo{}, // AuthInfos is a map of referencable names to user configs
		Contexts:   map[string]*api.Context{},  // Contexts is a map of referencable names to context configs
	}

	gen, err := token.NewGenerator(true, false)
	if err != nil {
		return nil, err
	}

	opts := &token.GetTokenOptions{
		ClusterID: *eksclusterName,
		Session:   sess,
	}
	token, err := gen.GetWithOptions(opts)
	if err != nil {
		return nil, err
	}

	// example: eks_eu-central-1_cluster-1 => https://XX.XX.XX.XX
	name := fmt.Sprintf("eks_%s_%s", region, *eksclusterName)

	cert, err := base64.StdEncoding.DecodeString(aws.StringValue(cluster.CertificateAuthority.Data))
	if err != nil {
		return nil, err
	}

	config.Clusters[name] = &api.Cluster{
		CertificateAuthorityData: cert,
		Server:                   *cluster.Endpoint,
	}
	config.CurrentContext = name

	// Just reuse the context name as an auth name.
	config.Contexts[name] = &api.Context{
		Cluster:  name,
		AuthInfo: name,
	}
	// AWS specific configation; use cloud platform scope.
	config.AuthInfos[name] = &api.AuthInfo{
		Token: token.Token,
	}
	return &config, nil
}

func getAWSSession(accessKeyID, secretAccessKey, region string) (*session.Session, error) {
	config := aws.
		NewConfig().
		WithRegion(region).
		WithCredentials(credentials.NewStaticCredentials(accessKeyID, secretAccessKey, "")).
		WithMaxRetries(3)

	sess, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API session: %v", err)
	}

	return sess, nil
}
