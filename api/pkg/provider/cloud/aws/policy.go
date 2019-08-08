package aws

import (
	"bytes"
	"text/template"
)

var (
	// This allows instances to perform the assume role
	assumeRolePolicy = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": { "Service": "ec2.amazonaws.com"},
      "Action": "sts:AssumeRole"
    }
  ]
}
`
	// The role for worker nodes.
	// Based on https://github.com/kubernetes/kops/blob/master/docs/iam_roles.md
	// Both actions cannot be restricted by tag filtering.
	workerRolePolicy = `{
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
    }
  ]
}
`

	// The role for control plane.
	// Based on https://github.com/kubernetes/kops/blob/master/docs/iam_roles.md
	// We're using 2 filters:
	// - RequestTag  = Makes sure the tag exists on the create request
	// - ResourceTag = Makes sure the tag exists on the resource that should be modified
	// Actions are grouped by ability of filtering:
	// - Actions which cannot be filtered (Setting a filter will make them fail)
	// - EC2 create actions with the RequestTag filter
	// - EC2 modify actions with the ResourceTag filter
	// - ELB create actions with the RequestTag filter
	// - ELB modify actions with the ResourceTag filter
	controlPlanePolicyTpl = template.Must(template.New("worker-policy").Parse(`{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
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
        "ec2:DescribeVpcs"
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
        "elasticloadbalancing:DescribeLoadBalancers",
        "elasticloadbalancing:DescribeLoadBalancerAttributes",
        "elasticloadbalancing:DetachLoadBalancerFromSubnets",
        "elasticloadbalancing:DeregisterInstancesFromLoadBalancer",
        "elasticloadbalancing:ModifyLoadBalancerAttributes",
        "elasticloadbalancing:RegisterInstancesWithLoadBalancer",
        "elasticloadbalancing:SetLoadBalancerPoliciesForBackendServer",
        "elasticloadbalancing:AddTags",
        "elasticloadbalancing:DeleteListener",
        "elasticloadbalancing:DeleteTargetGroup",
        "elasticloadbalancing:DeregisterTargets",
        "elasticloadbalancing:DescribeListeners",
        "elasticloadbalancing:DescribeLoadBalancerPolicies",
        "elasticloadbalancing:DescribeTargetGroups",
        "elasticloadbalancing:DescribeTargetHealth",
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

func getControlPlanePolicy(clusterName string) (string, error) {
	tag := clusterTag(clusterName)

	buf := &bytes.Buffer{}
	err := controlPlanePolicyTpl.Execute(buf, policyTplData{ClusterTag: *tag.Key})
	return buf.String(), err
}

type policyTplData struct {
	ClusterTag string
}
