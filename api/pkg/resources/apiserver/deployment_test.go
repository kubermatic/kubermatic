package apiserver

import (
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	"k8s.io/apimachinery/pkg/util/sets"
)

func TestGetApiserverFlags(t *testing.T) {
	tests := []struct {
		name              string
		kubernetesVersion string
		expectedFlags     []string
	}{
		{
			name:              "Ensure_no_admission_webhooks_pre_1.9",
			kubernetesVersion: "1.8.0",
			expectedFlags: []string{"--admission-control",
				"NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,ResourceQuota"},
		},
		{
			name:              "Ensure_admission_webhooks_1.9+",
			kubernetesVersion: "1.9.0",
			expectedFlags: []string{"--admission-control",
				"NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota"},
		},
		{
			name:              "Ensure_new_admission_flagname_1.10.0",
			kubernetesVersion: "1.10.0",
			expectedFlags: []string{"--enable-admission-plugins",
				"NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota"},
		},
	}

	for _, test := range tests {
		templateData := resources.TemplateData{}
		templateData.Cluster = &kubermaticv1.Cluster{}
		templateData.Cluster.Spec.ClusterNetwork.Services.CIDRBlocks = []string{"10.0.0.0"}
		templateData.Cluster.Spec.Cloud = &kubermaticv1.CloudSpec{}
		templateData.Cluster.Spec.Version = test.kubernetesVersion

		flags := getApiserverFlags(&templateData, 0, []string{"etcd-0"})
		flagSet := sets.NewString(flags...)

		for _, expectedFlag := range test.expectedFlags {
			if !flagSet.Has(expectedFlag) {
				t.Errorf("Expected flag '%s' in test '%s' but was not present!", expectedFlag, test.name)
			}
		}
	}

}
