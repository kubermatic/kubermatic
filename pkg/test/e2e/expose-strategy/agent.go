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

package exposestrategy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/nodeportproxy"
	envoyagent "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/envoy-agent"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	e2eutils "k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	agentDeployTimeout = 10 * time.Minute
)

// AgentConfig contains the configuration to deploy an agent like the ones
// deployed on each node when Tunneling strategy is used.
type AgentConfig struct {
	Log       *zap.SugaredLogger
	Namespace string
	Client    ctrlruntimeclient.Client
	Versions  kubermatic.Versions

	// AgentPod is an agent pod using Envoy image.
	AgentPod       *corev1.Pod
	AgentConfigMap *corev1.ConfigMap
}

// DeployAgentPod deploys the pod to be used to verify tunneling expose strategy.
func (a *AgentConfig) DeployAgentPod(ctx context.Context) error {
	agentCm := a.newAgentConfigMap(a.Namespace)
	if err := a.Client.Create(ctx, agentCm); err != nil {
		return fmt.Errorf("failed to create agent config map: %w", err)
	}
	a.AgentConfigMap = agentCm
	agentPod := a.newAgentPod(a.Namespace)
	if err := a.Client.Create(ctx, agentPod); err != nil {
		return fmt.Errorf("failed to create agent pod: %w", err)
	}

	if !e2eutils.CheckPodsRunningReady(ctx, a.Client, a.Log, a.Namespace, []string{agentPod.Name}, agentDeployTimeout) {
		return errors.New("timeout occurred while waiting for agent pod readiness")
	}

	if err := a.Client.Get(ctx, ctrlruntimeclient.ObjectKey{
		Namespace: agentPod.Namespace,
		Name:      agentPod.Name,
	}, agentPod); err != nil {
		return fmt.Errorf("failed to get agent pod: %w", err)
	}
	a.AgentPod = agentPod
	return nil
}

// CleanUp deletes the resources.
func (a *AgentConfig) CleanUp(ctx context.Context) error {
	if a.AgentPod != nil {
		return a.Client.Delete(ctx, a.AgentPod)
	}
	if a.AgentConfigMap != nil {
		return a.Client.Delete(ctx, a.AgentConfigMap)
	}
	return nil
}

// GetKASHostPort returns the host:port that can be used to reach the KAS
// passing from the tunnel.
func (a *AgentConfig) GetKASHostPort() string {
	return net.JoinHostPort(a.AgentPod.Status.PodIP, "6443")
}

// newAgnhostPod returns a pod returns the manifest of the agent pod.
func (a *AgentConfig) newAgentConfigMap(ns string) *corev1.ConfigMap {
	cmName, createConfigMap := envoyagent.ConfigMapReconciler(envoyagent.Config{
		AdminPort: 9902,
		ProxyHost: fmt.Sprintf("%s.kubermatic.svc.cluster.local", nodeportproxy.ServiceName),
		ProxyPort: 8088,
		Listeners: []envoyagent.Listener{
			{
				BindAddress: "0.0.0.0",
				BindPort:    6443,
				Authority:   net.JoinHostPort(fmt.Sprintf("apiserver-external.%s.svc.cluster.local", ns), "443"),
			},
		},
	})()
	// TODO: errors should never be thrown here
	cm, _ := createConfigMap(&corev1.ConfigMap{})
	cm.Name = cmName
	cm.Namespace = ns
	return cm
}

// newAgnhostPod returns a pod returns the manifest of the agent pod.
func (a *AgentConfig) newAgentPod(ns string) *corev1.Pod {
	agentName, createDaemonSet := envoyagent.DaemonSetReconciler(nil, net.IPv4(0, 0, 0, 0), a.Versions, "", registry.GetImageRewriterFunc(""))()

	ds, err := createDaemonSet(&appsv1.DaemonSet{})
	if err != nil {
		panic(err)
	}

	// We don't need the init containers in this context.
	ds.Spec.Template.Spec.InitContainers = []corev1.Container{}
	// We don't use host network
	ds.Spec.Template.Spec.HostNetwork = false
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agentName,
			Namespace: ns,
		},
		Spec: ds.Spec.Template.Spec,
	}
	return pod
}
