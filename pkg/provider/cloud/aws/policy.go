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
	"bytes"
	"fmt"
	"text/template"
)

var (
	// This allows instances to perform the assume role.
	assumeRolePolicy = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": { "Service": "ec2.amazonaws.com"},
      "Action": "sts:AssumeRole"
    },
	{
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::{{ .AccountID }}:root"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
`
	// The role for worker nodes.
	// Based on https://github.com/kubernetes/kops/blob/master/docs/iam_roles.md
	// Both actions cannot be restricted by tag filtering.
	//
	// All but the first statement are based on the AWS EBS CSI Driver on
	// https://github.com/kubernetes-sigs/aws-ebs-csi-driver/blob/master/docs/example-iam-policy.json
	workerRolePolicyTpl = template.Must(template.New("worker-policy").Parse(`{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeInstances",
        "ec2:DescribeRegions"
      ],
      "Resource": [
        "*"
      ]
    },

    {
      "Effect": "Allow",
      "Action": [
        "ec2:CreateSnapshot",
        "ec2:AttachVolume",
        "ec2:DetachVolume",
        "ec2:ModifyVolume",
        "ec2:DescribeAvailabilityZones",
        "ec2:DescribeInstances",
        "ec2:DescribeSnapshots",
        "ec2:DescribeTags",
        "ec2:DescribeVolumes",
        "ec2:DescribeVolumesModifications"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:CreateTags"
      ],
      "Resource": [
        "arn:aws:ec2:*:*:volume/*",
        "arn:aws:ec2:*:*:snapshot/*"
      ],
      "Condition": {
        "StringEquals": {
          "ec2:CreateAction": [
            "CreateVolume",
            "CreateSnapshot"
          ]
        }
      }
    },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DeleteTags"
      ],
      "Resource": [
        "arn:aws:ec2:*:*:volume/*",
        "arn:aws:ec2:*:*:snapshot/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:CreateVolume"
      ],
      "Resource": "*",
      "Condition": {
        "StringLike": {
          "aws:RequestTag/{{ .EBSCSIClusterTag }}": "true"
        }
      }
    },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:CreateVolume"
      ],
      "Resource": "*",
      "Condition": {
        "StringLike": {
          "aws:RequestTag/CSIVolumeName": "*"
        }
      }
    },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:CreateVolume"
      ],
      "Resource": "*",
      "Condition": {
        "StringLike": {
          "aws:RequestTag/{{ .ClusterTag }}": "owned"
        }
      }
    },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DeleteVolume"
      ],
      "Resource": "*",
      "Condition": {
        "StringLike": {
          "ec2:ResourceTag/{{ .EBSCSIClusterTag }}": "true"
        }
      }
    },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DeleteVolume"
      ],
      "Resource": "*",
      "Condition": {
        "StringLike": {
          "ec2:ResourceTag/CSIVolumeName": "*"
        }
      }
    },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DeleteVolume"
      ],
      "Resource": "*",
      "Condition": {
        "StringLike": {
          "ec2:ResourceTag/{{ .ClusterTag }}": "owned"
        }
      }
    },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DeleteSnapshot"
      ],
      "Resource": "*",
      "Condition": {
        "StringLike": {
          "ec2:ResourceTag/CSIVolumeSnapshotName": "*"
        }
      }
    },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DeleteSnapshot"
      ],
      "Resource": "*",
      "Condition": {
        "StringLike": {
          "ec2:ResourceTag/{{ .EBSCSIClusterTag }}": "true"
        }
      }
    }
  ]
}
`))

	// The role for control plane.
	// Based on https://github.com/kubernetes/kops/blob/master/docs/iam_roles.md and
	// https://github.com/kubernetes/cloud-provider-aws/blob/master/docs/prerequisites.md#iam-policies
	// We're using 2 filters:
	// - RequestTag  = Makes sure the tag exists on the create request
	// - ResourceTag = Makes sure the tag exists on the resource that should be modified
	// Actions are grouped by ability of filtering:
	// - Actions which cannot be filtered (Setting a filter will make them fail)
	// - EC2 create actions with the RequestTag filter
	// - EC2 modify actions with the ResourceTag filter
	// - ELB create actions with the RequestTag filter
	// - ELB modify actions with the ResourceTag filter.
	controlPlanePolicyTpl = template.Must(template.New("control-plane-policy").Parse(`{
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Action": [
		  "ec2:DescribeAvailabilityZones",
          "ec2:DescribeInstances",
          "ec2:DescribeRegions",
          "ec2:DescribeRouteTables",
          "ec2:DescribeSecurityGroups",
          "ec2:DescribeSubnets",
          "ec2:DescribeVolumes",
          "ec2:CreateSecurityGroup",
          "ec2:DescribeVolumesModifications",
          "ec2:ModifyInstanceAttribute",
          "ec2:ModifyVolume",
          "ec2:DescribeVpcs",
          "elasticloadbalancing:DescribeLoadBalancers",
          "elasticloadbalancing:DescribeLoadBalancerAttributes",
          "elasticloadbalancing:DescribeListeners",
          "elasticloadbalancing:DescribeLoadBalancerPolicies",
          "elasticloadbalancing:DescribeTargetGroups",
          "elasticloadbalancing:DescribeTargetHealth"
        ],
        "Resource": [
          "*"
        ]
      },
      {
        "Effect": "Allow",
        "Action": [
          "ec2:CreateVolume",
          "ec2:CreateRoute"
        ],
        "Resource": [
          "*"
        ],
        "Condition": {
          "Null": {
            "ec2:RequestTag/{{ .ClusterTag }}": "false"
          }
        }
      },
      {
        "Effect": "Allow",
        "Action": [
          "ec2:CreateTags",
          "ec2:AttachVolume",
          "ec2:AuthorizeSecurityGroupIngress",
          "ec2:DeleteRoute",
          "ec2:DeleteSecurityGroup",
          "ec2:DeleteVolume",
          "ec2:DetachVolume",
          "ec2:RevokeSecurityGroupIngress"
        ],
        "Resource": [
          "*"
        ],
        "Condition": {
          "Null": {
            "ec2:ResourceTag/{{ .ClusterTag }}": "false"
          }
        }
      },
	  {
	    "Effect": "Allow",
	    "Action": [
		  "ec2:CreateTags"
	    ],
	    "Resource": [
		  "arn:aws:ec2:*:*:security-group/*"
	    ],
	    "Condition": {
		  "StringEquals": {
		    "ec2:CreateAction": "CreateSecurityGroup"
		  }
	    }
	  },
      {
        "Effect": "Allow",
        "Action": [
          "elasticloadbalancing:CreateLoadBalancer",
          "elasticloadbalancing:CreateLoadBalancerPolicy",
          "elasticloadbalancing:CreateLoadBalancerListeners",
          "elasticloadbalancing:CreateListener",
          "elasticloadbalancing:CreateTargetGroup"
        ],
        "Resource": [
          "*"
        ],
        "Condition": {
          "Null": {
            "aws:RequestTag/{{ .ClusterTag }}": "false"
          }
        }
      },
      {
        "Effect": "Allow",
        "Action": [
          "elasticloadbalancing:AddTags",
          "elasticloadbalancing:AttachLoadBalancerToSubnets",
          "elasticloadbalancing:ApplySecurityGroupsToLoadBalancer",
          "elasticloadbalancing:ConfigureHealthCheck",
          "elasticloadbalancing:DeleteLoadBalancer",
          "elasticloadbalancing:DeleteLoadBalancerListeners",
          "elasticloadbalancing:DetachLoadBalancerFromSubnets",
          "elasticloadbalancing:DeregisterInstancesFromLoadBalancer",
          "elasticloadbalancing:ModifyLoadBalancerAttributes",
          "elasticloadbalancing:RegisterInstancesWithLoadBalancer",
          "elasticloadbalancing:SetLoadBalancerPoliciesForBackendServer",
          "elasticloadbalancing:DeleteListener",
          "elasticloadbalancing:DeleteTargetGroup",
          "elasticloadbalancing:DeregisterTargets",
          "elasticloadbalancing:ModifyListener",
          "elasticloadbalancing:ModifyTargetGroup",
          "elasticloadbalancing:RegisterTargets",
          "elasticloadbalancing:SetLoadBalancerPoliciesOfListener"
        ],
        "Resource": [
          "*"
        ],
        "Condition": {
          "Null": {
            "aws:ResourceTag/{{ .ClusterTag }}": "false"
          }
        }
      }
    ]
  }
  `))
)

func getWorkerPolicy(clusterName string) (string, error) {
	return renderPolicy(clusterName, workerRolePolicyTpl)
}

func getControlPlanePolicy(clusterName string) (string, error) {
	return renderPolicy(clusterName, controlPlanePolicyTpl)
}

func getAssumeRolePolicy(accountID string) (string, error) {
	assumeRolePolicyTpl := template.Must(template.New("assume-role-policy").Parse(assumeRolePolicy))

	buf := &bytes.Buffer{}
	err := assumeRolePolicyTpl.Execute(buf, assumeRoleTplData{
		AccountID: accountID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to render assume role policy: %w", err)
	}

	return buf.String(), nil
}

func renderPolicy(clusterName string, tpl *template.Template) (string, error) {
	tag := ec2ClusterTag(clusterName)

	buf := &bytes.Buffer{}
	err := tpl.Execute(buf, policyTplData{
		ClusterTag:       *tag.Key,
		EBSCSIClusterTag: fmt.Sprintf("ebs.csi.aws.com/%s", clusterName),
	})

	return buf.String(), err
}

type assumeRoleTplData struct {
	AccountID string
}

type policyTplData struct {
	ClusterTag       string
	EBSCSIClusterTag string
}
