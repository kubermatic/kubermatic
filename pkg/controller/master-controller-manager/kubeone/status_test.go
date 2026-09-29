/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package kubeone

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/machine-controller/sdk/providerconfig"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestKubeOneErrorMessage(t *testing.T) {
	long := strings.Repeat("x", maxKubeOneErrorMessageLength+10) + " the actual error"

	testCases := []struct {
		name     string
		raw      string
		expected string
	}{
		{
			name:     "multi-line KubeOne error",
			raw:      "Error: ssh: dialing\nconnection to: 10.0.0.5:22\ndial tcp 10.0.0.5:22: i/o timeout\n",
			expected: "ssh: dialing connection to: 10.0.0.5:22 dial tcp 10.0.0.5:22: i/o timeout",
		},
		{
			name:     "empty",
			raw:      " \n",
			expected: "",
		},
		{
			name:     "long message keeps the end",
			raw:      long,
			expected: "..." + string([]rune(long)[len([]rune(long))-maxKubeOneErrorMessageLength:]),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := kubeOneErrorMessage(tc.raw); got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestImportStatusMessage(t *testing.T) {
	testCases := []struct {
		name            string
		condition       kubermaticv1.ExternalClusterCondition
		lastImportError string
		expected        string
	}{
		{
			name:      "first attempt",
			condition: kubermaticv1.ExternalClusterCondition{Phase: kubermaticv1.ExternalClusterPhaseProvisioning},
			expected:  "trying to fetch cluster test kubeconfig",
		},
		{
			name:            "failed pod of the current job",
			condition:       kubermaticv1.ExternalClusterCondition{Phase: kubermaticv1.ExternalClusterPhaseProvisioning},
			lastImportError: "ssh: dialing",
			expected:        "import failed, retrying: ssh: dialing",
		},
		{
			name:      "error of the previous job",
			condition: kubermaticv1.ExternalClusterCondition{Phase: kubermaticv1.ExternalClusterPhaseSSHError, Message: "ssh: dialing"},
			expected:  "import failed, retrying: ssh: dialing",
		},
		{
			name:      "error is kept while the new job runs",
			condition: kubermaticv1.ExternalClusterCondition{Phase: kubermaticv1.ExternalClusterPhaseProvisioning, Message: "import failed, retrying: ssh: dialing"},
			expected:  "import failed, retrying: ssh: dialing",
		},
		{
			name:      "generic error is not treated as an import error",
			condition: kubermaticv1.ExternalClusterCondition{Phase: kubermaticv1.ExternalClusterPhaseError, Message: "missing ssh secret"},
			expected:  "trying to fetch cluster test kubeconfig",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cluster := &kubermaticv1.ExternalCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Status:     kubermaticv1.ExternalClusterStatus{Condition: tc.condition},
			}

			if got := importStatusMessage(cluster, tc.lastImportError); got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func failedPod(name string, finishedAt time.Time, message string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "kubeone-test", Labels: map[string]string{JobNameLabel: KubeOneImportJob}},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "kubeone",
				State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
					ExitCode:   13,
					FinishedAt: metav1.NewTime(finishedAt),
					Message:    message,
				}},
			}},
		},
	}
}

func TestLastFailedJobPodError(t *testing.T) {
	now := time.Now()
	client := ctrlruntimefakeclient.NewClientBuilder().
		WithIndex(&corev1.Pod{}, podPhaseKey, func(obj ctrlruntimeclient.Object) []string {
			return []string{string(obj.(*corev1.Pod).Status.Phase)}
		}).
		WithObjects(
			failedPod("older", now.Add(-time.Minute), "Error: configuration validation"),
			failedPod("newer", now, "Error: ssh: dialing\ndial tcp 10.0.0.5:22: i/o timeout"),
		).
		Build()

	r := &reconciler{Client: client, log: zap.NewNop().Sugar()}

	got, err := r.lastFailedJobPodError(context.Background(), "kubeone-test", KubeOneImportJob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expected := "ssh: dialing dial tcp 10.0.0.5:22: i/o timeout"; got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

// TestKubeOneScriptsReportFinalError runs the action part of the generated scripts with a stub
// kubeone, to ensure that the exit code is preserved, the import keeps stdout for the kubeconfig
// and the final KubeOne error is written to the termination message.
func TestKubeOneScriptsReportFinalError(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh is not available")
	}

	stub := "#!/bin/sh\necho 'kubeconfig-content'\necho 'time=\"12:00:00\" level=warning msg=\"Retrying task...\"' >&2\nprintf 'Error: ssh: dialing\\ndial tcp 10.0.0.5:22: i/o timeout\\n' >&2\nexit 13\n"

	for _, action := range []string{ImportAction, UpgradeControlPlaneAction, MigrateContainerRuntimeAction} {
		t.Run(action, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "kubeone"), []byte(stub), 0o755); err != nil {
				t.Fatal(err)
			}
			terminationLog := filepath.Join(dir, "termination-log")
			if err := os.WriteFile(terminationLog, nil, 0o644); err != nil {
				t.Fatal(err)
			}

			script := strings.TrimPrefix(generatedKubeOneScript(t, action), resources.KubeOneScript)
			script = strings.ReplaceAll(script, "/dev/termination-log", terminationLog)
			script = strings.ReplaceAll(script, kubeOneOutputFile, filepath.Join(dir, "kubeone.log"))

			cmd := exec.CommandContext(t.Context(), "sh", "-c", script)
			cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"))
			stdout, err := cmd.Output()

			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 13 {
				t.Fatalf("expected exit code 13, got %v", err)
			}

			if action == ImportAction && string(stdout) != "kubeconfig-content\n" {
				t.Errorf("expected only the kubeconfig on stdout, got %q", stdout)
			}

			message, err := os.ReadFile(terminationLog)
			if err != nil {
				t.Fatal(err)
			}
			if expected := "Error: ssh: dialing\ndial tcp 10.0.0.5:22: i/o timeout\n"; string(message) != expected {
				t.Errorf("expected termination message %q, got %q", expected, message)
			}
		})
	}
}

// generatedKubeOneScript returns the script the controller stores for the given KubeOne job action.
func generatedKubeOneScript(t *testing.T, action string) string {
	t.Helper()

	ref := func(name string) *providerconfig.GlobalSecretKeySelector {
		return &providerconfig.GlobalSecretKeySelector{
			ObjectReference: corev1.ObjectReference{Name: name, Namespace: resources.KubermaticNamespace},
		}
	}

	cluster := &kubermaticv1.ExternalCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "script"},
		Spec: kubermaticv1.ExternalClusterSpec{
			CloudSpec: kubermaticv1.ExternalClusterCloudSpec{
				KubeOne: &kubermaticv1.ExternalClusterKubeOneCloudSpec{
					ProviderName:         resources.KubeOneVSphere,
					CredentialsReference: ref("credentials"),
					SSHReference:         ref("ssh"),
					ManifestReference:    ref("manifest"),
				},
			},
		},
	}

	client := ctrlruntimefakeclient.NewClientBuilder().WithObjects(
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: resources.KubermaticNamespace}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ssh", Namespace: resources.KubermaticNamespace}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "manifest", Namespace: resources.KubermaticNamespace}},
	).Build()

	r := &reconciler{Client: client, log: zap.NewNop().Sugar()}
	job, err := r.generateKubeOneActionJob(context.Background(), r.log, resources.NewTemplateDataBuilder().Build(), cluster, action)
	if err != nil {
		t.Fatalf("failed to generate kubeone job: %v", err)
	}

	var configMapName string
	for _, volume := range job.Spec.Template.Spec.Volumes {
		if volume.ConfigMap != nil {
			configMapName = volume.ConfigMap.Name
		}
	}

	configMap := &corev1.ConfigMap{}
	if err := client.Get(context.Background(), types.NamespacedName{Name: configMapName, Namespace: cluster.GetKubeOneNamespaceName()}, configMap); err != nil {
		t.Fatalf("failed to get the script ConfigMap: %v", err)
	}

	return configMap.Data["script.sh"]
}
