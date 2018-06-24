package apiserver

import (
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
)

func TestGetAdmissionControlFlags(t *testing.T) {
	tests := []struct {
		name                              string
		kubernetesVersion                 string
		expectedAdmissionControlFlagName  string
		expectedAdmissionControlFlagValue string
	}{
		{
			name:                              "Ensure no admission webhooks pre 1.9",
			kubernetesVersion:                 "1.8.0",
			expectedAdmissionControlFlagName:  "--admission-control",
			expectedAdmissionControlFlagValue: "NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,ResourceQuota",
		},
		{
			name:                              "Ensure admission webhooks 1.9+",
			kubernetesVersion:                 "1.9.0",
			expectedAdmissionControlFlagName:  "--admission-control",
			expectedAdmissionControlFlagValue: "NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota",
		},
		{
			name:                              "Ensure new admission flagname 1.10+",
			kubernetesVersion:                 "1.10.0",
			expectedAdmissionControlFlagName:  "--enable-admission-plugins",
			expectedAdmissionControlFlagValue: "NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota",
		},
	}

	for _, test := range tests {
		templateData := resources.TemplateData{}
		templateData.Cluster = &kubermaticv1.Cluster{}
		templateData.Cluster.Spec.Version = test.kubernetesVersion

		admissionControlFlagName, admissionControlFlagValue := getAdmissionControlFlags(&templateData)
		if admissionControlFlagName != test.expectedAdmissionControlFlagName {
			t.Errorf("Expected admission control flag name to be %s but was %s", test.expectedAdmissionControlFlagName, admissionControlFlagName)
		}

		if admissionControlFlagValue != test.expectedAdmissionControlFlagValue {
			t.Errorf("Expected admission control flag value to be %s but was %s", test.expectedAdmissionControlFlagValue, admissionControlFlagValue)
		}

	}

}
