package scheduler

import (
	"bytes"
	"fmt"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	corev1 "k8s.io/api/core/v1"
	"text/template"
)

type schedulerTplModel struct {
	TemplateData         resources.TemplateData
	PolicyConfigFileName string
}

const (
	schedulerConfigTpl = `{{ if lt .TemplateData.Cluster.Spec.Version.Semver.Minor 12 -}}
apiVersion: componentconfig/v1alpha1
{{- else -}}
apiVersion: kubescheduler.config.k8s.io/v1alpha1
{{- end }}
kind: KubeSchedulerConfiguration
algorithmSource:
  policy:
    file:
      path: /etc/kubernetes/scheduler/{{ .PolicyConfigFileName }}
clientConnection:
  kubeconfig: /etc/kubernetes/kubeconfig/kubeconfig
`

	schedulerPolicyTpl = `{
    "apiVersion": "v1",
    "kind": "Policy",
    "predicates": [
{{- if .TemplateData.Cluster.Spec.Cloud.Azure }}
        {
            "name": "MaxAzureDiskVolumeCount"
        },
{{- end }}
        {
            "name": "MatchInterPodAffinity"
        },
        {
            "name": "GeneralPredicates"
        },
        {
            "name": "CheckVolumeBinding"
        },
        {
            "name": "CheckNodeUnschedulable"
        },
{{- if .TemplateData.Cluster.Spec.Cloud.AWS }}
        {
            "name": "MaxEBSVolumeCount"
        },
{{- end }}
{{- if false }}
        {
            "name": "MaxGCEPDVolumeCount"
        },
{{- end }}
        {
            "name": "NoDiskConflict"
        },
        {
            "name": "NoVolumeZoneConflict"
        },
        {
            "name": "MaxCSIVolumeCountPred"
        },
{{- if and .TemplateData.Cluster.Spec.Cloud.Openstack (ge .TemplateData.Cluster.Spec.Version.Semver.Minor 14) }}
        {
            "name": "MaxCinderVolumeCount"
        },
{{- end }}
        {
            "name": "PodToleratesNodeTaints"
        }
    ],
    "priorities": [
        {
            "name": "SelectorSpreadPriority",
            "weight": 1
        },
        {
            "name": "InterPodAffinityPriority",
            "weight": 1
        },
        {
            "name": "LeastRequestedPriority",
            "weight": 1
        },
        {
            "name": "BalancedResourceAllocation",
            "weight": 1
        },
        {
            "name": "ImageLocalityPriority",
            "weight": 1
        },
        {
            "name": "NodePreferAvoidPodsPriority",
            "weight": 10000
        },
        {
            "name": "NodeAffinityPriority",
            "weight": 1
        },
        {
            "name": "TaintTolerationPriority",
            "weight": 1
        }
    ]
}`
)

// ConfigMapCreator returns a function to create the ConfigMap containing the scheduler's configuration files
func ConfigMapCreator(data *resources.TemplateData) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return resources.SchedulerConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}

			schedulerConfig, err := ExpandTemplate(schedulerConfigTpl, data)
			if err != nil {
				return nil, fmt.Errorf("failed to create scheduler config: %v", err)
			}

			policyConfig, err := ExpandTemplate(schedulerPolicyTpl, data)
			if err != nil {
				return nil, fmt.Errorf("failed to create scheduler policy: %v", err)
			}

			cm.Labels = resources.BaseAppLabel(name, nil)
			cm.Data[resources.SchedulerConfigFileName] = schedulerConfig
			cm.Data[resources.SchedulerPolicyFileName] = policyConfig

			return cm, nil
		}
	}
}

func ExpandTemplate(templateText string, data *resources.TemplateData) (expansion string, err error) {
	tpl, err := template.New("scheduler").Parse(templateText)
	if err != nil {
		return "", fmt.Errorf("failed to parse the scheduler policy template: %v", err)
	}

	model := &schedulerTplModel{
		TemplateData:         *data,
		PolicyConfigFileName: resources.SchedulerPolicyFileName,
	}

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, model); err != nil {
		return "", fmt.Errorf("failed to execute scheduler policy template: %v", err)
	}

	return buf.String(), nil
}
