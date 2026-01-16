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

package utils

import (
	"context"
	"errors"
	"fmt"
	"time"

	constrainttemplatev1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/podexec"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/util/podutils"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	gwapischeme "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/scheme"
)

const (
	// pollPeriod is how often to poll pods, nodes and claims.
	pollPeriod = 2 * time.Second
)

type TestPodConfig struct {
	Log       *zap.SugaredLogger
	Namespace string
	Client    ctrlruntimeclient.Client
	Config    *rest.Config
	// CreatePodFunc returns the manifest of the pod to be used for running the
	// test. As we need to exec the pod should not terminate (e.g. run an
	// infinite sleep).
	CreatePodFunc       func(ns string) *corev1.Pod
	PodReadinessTimeout time.Duration

	testPod *corev1.Pod
}

// DeployTestPod deploys the pod to be used to run the test command.
func (t *TestPodConfig) DeployTestPod(ctx context.Context, log *zap.SugaredLogger) error {
	testPod := t.CreatePodFunc(t.Namespace)
	if err := t.Client.Create(ctx, testPod); err != nil {
		return fmt.Errorf("failed to create host test Pod: %w", err)
	}

	// Use default timeout of 5 minutes if not otherwise specified.
	if t.PodReadinessTimeout == 0 {
		t.PodReadinessTimeout = 5 * time.Minute
	}
	if !CheckPodsRunningReady(ctx, t.Client, log, t.Namespace, []string{testPod.Name}, t.PodReadinessTimeout) {
		return errors.New("timeout occurred while waiting for host test Pod readiness")
	}

	if err := t.Client.Get(ctx, ctrlruntimeclient.ObjectKey{
		Namespace: testPod.Namespace,
		Name:      testPod.Name,
	}, testPod); err != nil {
		return fmt.Errorf("failed to get host test pod: %w", err)
	}
	t.testPod = testPod
	return nil
}

// CleanUp deletes the resources.
func (t *TestPodConfig) CleanUp(ctx context.Context) error {
	if t.testPod != nil {
		return t.Client.Delete(ctx, t.testPod)
	}
	return nil
}

// Exec executes the given command in the chosen container of the test pod.
func (t *TestPodConfig) Exec(ctx context.Context, container string, command ...string) (string, string, error) {
	if t.testPod == nil {
		return "", "", errors.New("exec should be called only after successful DeployTestPod execution")
	}

	return podexec.ExecuteCommand(ctx, t.Config, types.NamespacedName{
		Namespace: t.Namespace,
		Name:      t.testPod.Name,
	}, container, command...)
}

// GetClientsOrDie returns the clients used for testing or panics if something
// goes wrong during the clients creation.
func GetClientsOrDie() (ctrlruntimeclient.Client, *rest.Config) {
	cli, config, err := GetClients()
	if err != nil {
		panic(err)
	}
	return cli, config
}

// GetClients returns the clients used for testing or an error if something
// goes wrong during the clients creation.
func GetClients() (ctrlruntimeclient.Client, *rest.Config, error) {
	sc := runtime.NewScheme()
	if err := scheme.AddToScheme(sc); err != nil {
		return nil, nil, err
	}

	if err := kubermaticv1.AddToScheme(sc); err != nil {
		return nil, nil, err
	}

	if err := constrainttemplatev1.AddToScheme(sc); err != nil {
		return nil, nil, err
	}

	if err := gwapischeme.AddToScheme(sc); err != nil {
		return nil, nil, err
	}

	config, err := ctrlruntime.GetConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get kube config: %w", err)
	}

	httpClient, err := rest.HTTPClientFor(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	mapper, err := apiutil.NewDynamicRESTMapper(config, httpClient)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create dynamic REST mapper: %w", err)
	}

	c, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{
		Mapper: mapper,
		Scheme: sc,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create client: %w", err)
	}

	return c, config, nil
}

// WaitForPodsCreated waits for the given replicas number of pods matching the
// given set of labels to be created, and returns the names of the matched
// pods.
func WaitForPodsCreated(ctx context.Context, c ctrlruntimeclient.Client, log *zap.SugaredLogger, replicas int, namespace string, matchLabels map[string]string) ([]string, error) {
	timeout := 2 * time.Minute
	listOpts := []ctrlruntimeclient.ListOption{
		ctrlruntimeclient.InNamespace(namespace),
		ctrlruntimeclient.MatchingLabels(matchLabels),
	}

	logger := log.With("namespace", namespace)
	logger.With("timeout", timeout).Infof("Waiting for %d Pod(s) to be created...", replicas)

	// List the pods, making sure we observe all the replicas.
	foundPods := []string{}

	err := wait.PollImmediateLog(ctx, logger, 2*time.Second, timeout, func(ctx context.Context) (transient error, terminal error) {
		pods := corev1.PodList{}
		if err := c.List(ctx, &pods, listOpts...); err != nil {
			return nil, fmt.Errorf("failed to list Pods: %w", err)
		}

		foundPods = []string{}
		for _, pod := range pods.Items {
			if pod.DeletionTimestamp != nil {
				continue
			}
			foundPods = append(foundPods, pod.Name)
		}

		if len(foundPods) >= replicas {
			return nil, nil
		}

		return fmt.Errorf("found %d/%d Pods", len(foundPods), replicas), nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to wait for Pods: %w", err)
	}

	logger.Info("Found all expected Pods")

	return foundPods, err
}

// CheckPodsRunningReady returns whether all pods whose names are listed in
// podNames in namespace ns are running and ready, using c and waiting at most
// timeout.
func CheckPodsRunningReady(ctx context.Context, c ctrlruntimeclient.Client, log *zap.SugaredLogger, ns string, podNames []string, timeout time.Duration) bool {
	condition := func(pod *corev1.Pod) (error, error) {
		return PodRunningReady(pod), nil
	}

	return checkPodsCondition(ctx, c, log, ns, podNames, timeout, condition, "running and ready")
}

type podCondition func(pod *corev1.Pod) (transient error, terminal error)

// WaitForPodCondition waits a pods to be matched to the given condition.
func WaitForPodCondition(ctx context.Context, c ctrlruntimeclient.Client, log *zap.SugaredLogger, ns, podName, desc string, timeout time.Duration, condition podCondition) error {
	key := ctrlruntimeclient.ObjectKey{Name: podName, Namespace: ns}

	// namespace and timeout are already set in the log's context
	logger := log.With("pod", podName)
	logger.With("timeout", timeout).Infof("Waiting for Pod to be %q...", desc)

	return wait.PollImmediateLog(ctx, logger, pollPeriod, timeout, func(ctx context.Context) (error, error) {
		pod := corev1.Pod{}
		if err := c.Get(ctx, key, &pod); err != nil {
			if apierrors.IsNotFound(err) {
				return nil, err
			}
			return fmt.Errorf("failed to get Pod: %w", err), nil
		}

		logger.Debugw(
			"Pod status",
			"phase", pod.Status.Phase,
			"reason", pod.Status.Reason,
			"ready", podutils.IsPodReady(&pod),
		)

		return condition(&pod)
	})
}

// checkPodsCondition returns whether all pods whose names are listed in podNames
// in namespace ns are in the condition, using c and waiting at most timeout.
func checkPodsCondition(ctx context.Context, c ctrlruntimeclient.Client, log *zap.SugaredLogger, ns string, podNames []string, timeout time.Duration, condition podCondition, desc string) bool {
	logger := log.With("namespace", ns, "timeout", timeout)
	logger.Infof("Waiting for Pods to be %q...", desc)

	type waitPodResult struct {
		success bool
		podName string
	}

	result := make(chan waitPodResult, len(podNames))
	for _, podName := range podNames {
		// Launch off pod readiness checkers.
		go func(name string) {
			err := WaitForPodCondition(ctx, c, logger, ns, name, desc, timeout, condition)
			result <- waitPodResult{err == nil, name}
		}(podName)
	}

	// Wait for them all to finish.
	success := true
	for range podNames {
		res := <-result
		if !res.success {
			logger.Errorw("Pod failed to reach desired condition.", "pod", res.podName)
			success = false
		}
	}

	return success
}

// PodRunningReady checks whether pod p's phase is running and it has a ready
// condition of status true.
func PodRunningReady(p *corev1.Pod) error {
	if p.Status.Phase != corev1.PodRunning {
		return fmt.Errorf("Pod is not %v, but %v", corev1.PodRunning, p.Status.Phase)
	}

	if !podutils.IsPodReady(p) {
		return fmt.Errorf("Pod is not ready, but: %v", p.Status.Conditions)
	}

	return nil
}

// WaitForDeploymentReady waits until the Deployment is fully ready.
func WaitForDeploymentReady(ctx context.Context, c ctrlruntimeclient.Client, log *zap.SugaredLogger, ns, name string, timeout time.Duration) error {
	key := ctrlruntimeclient.ObjectKey{Name: name, Namespace: ns}

	// namespace and timeout are already set in the log's context
	logger := log.With("deployment", key.String())
	logger.Info("Waiting for Deployment to be ready...")

	return wait.PollImmediateLog(ctx, log, 5*time.Second, timeout, func(ctx context.Context) (error, error) {
		status, err := resources.HealthyDeployment(ctx, c, key, -1)
		if err != nil {
			return nil, err
		}

		if status != kubermaticv1.HealthStatusUp {
			return fmt.Errorf("Deployment is %v", status), nil
		}

		return nil, nil
	})
}
