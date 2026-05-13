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

package usersshkeysagent

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	usersshkeysagent "k8c.io/kubermatic/v2/pkg/controller/usersshkeysagent"
	"k8c.io/kubermatic/v2/pkg/util/podexec"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	agentLabelKey      = "app"
	agentLabelValue    = "user-ssh-keys-agent"
	agentContainer     = "user-ssh-keys-agent"
	debugContainerName = "debug-cat"
)

// authorizedKeysView is the parsed form of an authorized_keys file: KKP-managed
// keys keyed by their UserSSHKey object name, plus external (unmarked) keys.
type authorizedKeysView = usersshkeysagent.AuthorizedKeysView

// generateSSHKey returns an OpenSSH-formatted ed25519 public key suitable for
// authorized_keys and providerSpec.sshPublicKeys.
func generateSSHKey(t *testing.T) string {
	t.Helper()

	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate ed25519 key: %v", err)
	}

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("failed to wrap public key: %v", err)
	}

	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub)))
}

// parseAuthorizedKeys parses an authorized_keys file using the same logic as
// the agent controller, so the e2e tests validate against the real parser.
func parseAuthorizedKeys(content string) authorizedKeysView {
	return usersshkeysagent.ParseAuthorizedKeys(content)
}

// findAgentPodForNode returns the user-ssh-keys-agent pod scheduled on the
// given node, or nil if none is found.
func findAgentPodForNode(ctx context.Context, userClient ctrlruntimeclient.Client, nodeName string) (*corev1.Pod, error) {
	pods := corev1.PodList{}

	err := userClient.List(ctx, &pods,
		ctrlruntimeclient.InNamespace(metav1.NamespaceSystem),
		ctrlruntimeclient.MatchingLabels{agentLabelKey: agentLabelValue},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent pods: %w", err)
	}

	for i := range pods.Items {
		if pods.Items[i].Spec.NodeName == nodeName {
			return &pods.Items[i], nil
		}
	}

	return nil, nil
}

// ensureDebugContainer attaches a busybox ephemeral container to the given pod
// for reading host files. The agent image is distroless and has no shell
// utilities, so we inject a debug container to get cat access.
// The ephemeral container shares the pod's volume mounts (/home, /root).
func ensureDebugContainer(ctx context.Context, cfg *rest.Config, pod *corev1.Pod) error {
	for _, ec := range pod.Spec.EphemeralContainers {
		if ec.Name == debugContainerName {
			return nil
		}
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	ec := corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:                     debugContainerName,
			Image:                    "busybox:1.36",
			ImagePullPolicy:          corev1.PullIfNotPresent,
			Command:                  []string{"sleep", "infinity"},
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			VolumeMounts: []corev1.VolumeMount{
				{Name: "root", MountPath: "/root"},
				{Name: "home", MountPath: "/home"},
			},
		},
		TargetContainerName: agentContainer,
	}

	pod.Spec.EphemeralContainers = append(pod.Spec.EphemeralContainers, ec)
	_, err = client.CoreV1().Pods(pod.Namespace).UpdateEphemeralContainers(ctx, pod.Name, pod, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to attach ephemeral debug container: %w", err)
	}

	// wait for the ephemeral container to be running.
	return wait.PollUntilContextTimeout(ctx, 2*time.Second, 2*time.Minute, false, func(ctx context.Context) (bool, error) {
		updated, err := client.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, s := range updated.Status.EphemeralContainerStatuses {
			if s.Name == debugContainerName && s.State.Running != nil {
				return true, nil
			}
		}

		return false, nil
	})
}

// readAuthorizedKeys execs into the ephemeral debug container attached to the
// agent pod and returns the contents of the host's authorized_keys file.
// The agent pod mounts /home and /root as hostPath volumes, making the node's
// SSH authorized_keys files visible inside the container.
func readAuthorizedKeys(ctx context.Context, cfg *rest.Config, pod *corev1.Pod, path string) (string, error) {
	if err := ensureDebugContainer(ctx, cfg, pod); err != nil {
		return "", fmt.Errorf("failed to ensure debug container: %w", err)
	}

	stdout, stderr, err := podexec.ExecuteCommand(
		ctx, cfg,
		types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name},
		debugContainerName,
		"cat", path,
	)
	if err != nil {
		return "", fmt.Errorf("cat %s failed (stdout=%q, stderr=%q): %w", path, stdout, stderr, err)
	}
	return stdout, nil
}

// newUserSSHKey builds (but does not create) a UserSSHKey CR attached to the
// given cluster and project.
func newUserSSHKey(name, displayName, projectName, clusterName, publicKey string) *kubermaticv1.UserSSHKey {
	return &kubermaticv1.UserSSHKey{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: kubermaticv1.SSHKeySpec{
			Name:      displayName,
			Project:   projectName,
			Clusters:  []string{clusterName},
			PublicKey: publicKey,
		},
	}
}
