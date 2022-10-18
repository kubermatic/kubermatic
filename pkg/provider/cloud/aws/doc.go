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

/*
Package aws contains the Amazon Web Services (AWS) cloud provider implementation.

This provider is responsible for managing a variety of resources in the
AWS cloud, some of which can pre-exist, some are always created explicitly
for a given usercluster. These resources are:

  - EC2: VPC
    A VPC must already exist. If the user doesn't specify a VPC, the provider chooses
    the default VPC for the given AWS account. If during reconciliation the VPC
    is not found, an error is raised and no further reconciliation can happen.

  - EC2: Route Table (RT)
    A usercluster can use its own RT, but if none is given by the user, the default
    RT for the VPC will be used (shared among many userclusters).
    KKP never creates or deletes route tables, it only tags them with the cluster tag.

  - EC2: Security Group (SG)
    This one can be specified by the user, but is otherwise created automatically.
    Every usercluster lives in its own SG and the SG is always tagged with the
    cluster tag.

  - EC2: Subnets
    The AWS CCM requires that all subnets are tagged with the cluster name, as
    documented in https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.1/deploy/subnet_discovery/.
    KKP does not create or remove subnets, but tags them, so that multiple userclusters
    can share the same subnets.

  - IAM: Control plane role
    This one can be specified by the user, but is otherwise created automatically.
    Every usercluster has its own control plan role. If the specified role does not
    exist, it is created.

  - IAM: Worker role & instance profile
    This one can be specified by the user, but is otherwise created automatically.
    Every usercluster has its own worker role/profile. If the specified profile does not
    exist, it is created.

During cluster deletion, KKP will try to clean up and remove unneeded resources again.
However, if the user specified a given field (e.g. a SG ID), KKP does not remove
the resource, assuming it is shared by either other userclusters or other things.
To keep track of ownership, an owner tag is placed on all resources that KKP creates.
The cluster tag for the AWS CCM is also removed, regardless whether the resource
was created by KKP or not.
*/
package aws
