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

package podexec

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// ExecuteCommand executes the given command in the chosen container of a pod.
func ExecuteCommand(ctx context.Context, restConfig *rest.Config, pod types.NamespacedName, container string, command ...string) (string, string, error) {
	gvk, err := apiutil.GVKForObject(&corev1.Pod{}, scheme.Scheme)
	if err != nil {
		return "", "", fmt.Errorf("failed to get pod GVK: %w", err)
	}

	httpClient, err := rest.HTTPClientFor(restConfig)
	if err != nil {
		return "", "", fmt.Errorf("failed to create HTTP client: %w", err)
	}

	restClient, err := apiutil.RESTClientForGVK(gvk, false, false, restConfig, scheme.Codecs, httpClient)
	if err != nil {
		return "", "", fmt.Errorf("failed to create REST client: %w", err)
	}

	request := restClient.
		Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("exec").
		Param("container", container)

	request.VersionedParams(&corev1.PodExecOptions{
		Container: container,
		Command:   command,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(restConfig, http.MethodPost, request.URL())
	if err != nil {
		return "", "", fmt.Errorf("failed to create SPDY executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return "", "", err
	}

	return stdout.String(), stderr.String(), err
}
