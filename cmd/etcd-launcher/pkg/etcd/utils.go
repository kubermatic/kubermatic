/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package etcd

import (
	"fmt"
	"io"

	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func closeClient(c io.Closer, log *zap.SugaredLogger) {
	err := c.Close()
	if err != nil {
		log.Warn(zap.Error(err))
	}
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func hasStrictTLS(pod *corev1.Pod) bool {
	for _, env := range pod.Spec.Containers[0].Env {
		if env.Name == "PEER_TLS_MODE" && env.Value == "strict" {
			return true
		}
	}

	return false
}

func inClusterClient() (ctrlruntimeclient.Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in cluster config: %w", err)
	}
	client, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster client: %w", err)
	}
	return client, nil
}
