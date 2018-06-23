package apiserver

import (
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
)

func TestGetAdmissionControlFlags(t *testing.T) {
	tests := []struct {
		name              string
		kubernetesVersion string
		expectedFlags     [2]string
	}{
		{
			name:              "Ensure no admission webhooks pre 1.9",
			kubernetesVersion: "1.8.0",
			expectedFlags: [2]string{"--admission-control",
				"NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,ResourceQuota"},
		},
		{
			name:              "Ensure admission webhooks 1.9+",
			kubernetesVersion: "1.9.0",
			expectedFlags: [2]string{"--admission-control",
				"NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota"},
		},
		{
			name:              "Ensure new admission flagname 1.10+",
			kubernetesVersion: "1.10.0",
			expectedFlags: [2]string{"--enable-admission-plugins",
				"NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota"},
		},
	}

	for _, test := range tests {
		templateData := resources.TemplateData{}
		templateData.Cluster = &kubermaticv1.Cluster{}
		templateData.Cluster.Spec.Version = test.kubernetesVersion

		admissionControlFlags := getAdmissionControlFlags(&templateData)

		for idx := range admissionControlFlags {
			if admissionControlFlags[idx] != test.expectedFlags[idx] {
				t.Errorf("Expected admission control flag to be %s but was %s", test.expectedFlags[idx], admissionControlFlags[idx])
			}
		}
	}

}
