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
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/util/podutils"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	constrainttemplatev1beta1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
)

const (
	// pollPeriod is how often to poll pods, nodes and claims.
	pollPeriod = 2 * time.Second
)

type TestPodConfig struct {
	Log           *zap.SugaredLogger
	Namespace     string
	Client        ctrlruntimeclient.Client
	PodRestClient rest.Interface
	Config        *rest.Config
	// CreatePodFunc returns the manifest of the pod to be used for running the
	// test. As we need to exec the pod should not terminate (e.g. run an
	// infinite sleep).
	CreatePodFunc       func(ns string) *corev1.Pod
	PodReadinessTimeout time.Duration

	testPod *corev1.Pod
}

// DeployTestPod deploys the pod to be used to run the test command.
func (t *TestPodConfig) DeployTestPod() error {
	testPod := t.CreatePodFunc(t.Namespace)
	if err := t.Client.Create(context.TODO(), testPod); err != nil {
		return errors.Wrap(err, "failed to create host test pod")
	}

	// Use default timeout of 5 minutes if not otherwise specified.
	if t.PodReadinessTimeout == 0 {
		t.PodReadinessTimeout = 5 * time.Minute
	}
	if !CheckPodsRunningReady(t.Client, t.Namespace, []string{testPod.Name}, t.PodReadinessTimeout) {
		return errors.New("timeout occurred while waiting for host test pod readiness")
	}

	if err := t.Client.Get(context.TODO(), ctrlruntimeclient.ObjectKey{
		Namespace: testPod.Namespace,
		Name:      testPod.Name,
	}, testPod); err != nil {
		return errors.Wrap(err, "failed to get host test pod")
	}
	t.testPod = testPod
	return nil
}

// CleanUp deletes the resources.
func (t *TestPodConfig) CleanUp() error {
	if t.testPod != nil {
		return t.Client.Delete(context.TODO(), t.testPod)
	}
	return nil
}

// Exec executes the given command in the chosen container of the test pod.
func (t *TestPodConfig) Exec(container string, command ...string) (string, string, error) {
	if t.testPod == nil {
		return "", "", errors.New("exec should be called only after successful DeployTestPod execution")
	}
	const tty = false

	req := t.PodRestClient.Post().
		Resource("pods").
		Name(t.testPod.Name).
		Namespace(t.Namespace).
		SubResource("exec").
		Param("container", container)
	req.VersionedParams(&corev1.PodExecOptions{
		Container: container,
		Command:   command,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       tty,
	}, scheme.ParameterCodec)

	var stdout, stderr bytes.Buffer
	err := execute("POST", req.URL(), t.Config, nil, &stdout, &stderr, tty)

	return stdout.String(), stderr.String(), err
}

func execute(method string, url *url.URL, config *rest.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool) error {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return err
	}
	return exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	})
}

// GetClientsOrDie returns the clients used for testing or panics if something
// goes wrong during the clients creation.
func GetClientsOrDie() (ctrlruntimeclient.Client, rest.Interface, *rest.Config) {
	cli, restCli, config, err := GetClients()
	if err != nil {
		panic(err)
	}
	return cli, restCli, config
}

// GetClients returns the clients used for testing or an error if something
// goes wrong during the clients creation.
func GetClients() (ctrlruntimeclient.Client, rest.Interface, *rest.Config, error) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, nil, nil, err
	}
	if err := kubermaticv1.AddToScheme(scheme); err != nil {
		return nil, nil, nil, err
	}
	if err := constrainttemplatev1beta1.AddToSchemes.AddToScheme(scheme); err != nil {
		return nil, nil, nil, err
	}

	config := ctrlruntime.GetConfigOrDie()
	mapper, err := apiutil.NewDynamicRESTMapper(config)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to create dynamic REST mapper")
	}
	gvk, err := apiutil.GVKForObject(&corev1.Pod{}, scheme)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to get pod GVK")
	}
	podRestClient, err := apiutil.RESTClientForGVK(gvk, false, config, serializer.NewCodecFactory(scheme))
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to create pod rest client")
	}
	c, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{
		Mapper: mapper,
		Scheme: scheme,
	})
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to create client")
	}
	return c, podRestClient, config, nil
}

// WaitForPodsCreated waits for the given replicas number of pods matching the
// given set of labels to be created, and returns the names of the matched
// pods.
func WaitForPodsCreated(c ctrlruntimeclient.Client, replicas int, namespace string, matchLabels map[string]string) ([]string, error) {
	timeout := 2 * time.Minute
	// List the pods, making sure we observe all the replicas.
	DefaultLogger.Debugf("Waiting up to %v for %d pods to be created", timeout, replicas)
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(2 * time.Second) {
		pods := corev1.PodList{}
		if err := c.List(context.TODO(), &pods,
			ctrlruntimeclient.InNamespace(namespace),
			ctrlruntimeclient.MatchingLabels(matchLabels)); err != nil {
			return nil, err
		}

		found := []string{}
		for _, pod := range pods.Items {
			if pod.DeletionTimestamp != nil {
				continue
			}
			found = append(found, pod.Name)
		}
		if len(found) == replicas {
			DefaultLogger.Infof("Found all %d pods", replicas)
			return found, nil
		}
		DefaultLogger.Debugf("Found %d/%d pods - will retry", len(found), replicas)
	}
	return nil, fmt.Errorf("timeout waiting for %d pods to be created", replicas)
}

// CheckPodsRunningReady returns whether all pods whose names are listed in
// podNames in namespace ns are running and ready, using c and waiting at most
// timeout.
func CheckPodsRunningReady(c ctrlruntimeclient.Client, ns string, podNames []string, timeout time.Duration) bool {
	return checkPodsCondition(c, ns, podNames, timeout, PodRunningReady, "running and ready")
}

// WaitForPodCondition waits a pods to be matched to the given condition.
func WaitForPodCondition(c ctrlruntimeclient.Client, ns, podName, desc string, timeout time.Duration, condition podCondition) error {
	DefaultLogger.Infof("Waiting up to %v for pod %q in namespace %q to be %q", timeout, podName, ns, desc)
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(pollPeriod) {
		pod := corev1.Pod{}
		if err := c.Get(context.TODO(), ctrlruntimeclient.ObjectKey{Name: podName, Namespace: ns}, &pod); err != nil {
			if apierrors.IsNotFound(err) {
				DefaultLogger.Debugf("Pod %q in namespace %q not found. Error: %v", podName, ns, err)
				return err
			}
			DefaultLogger.Debugf("Get pod %q in namespace %q failed, ignoring for %v. Error: %v", podName, ns, pollPeriod, err)
			continue
		}
		// log now so that current pod info is reported before calling `condition()`
		DefaultLogger.Debugf("Pod %q: Phase=%q, Reason=%q, readiness=%t. Elapsed: %v",
			podName, pod.Status.Phase, pod.Status.Reason, podutils.IsPodReady(&pod), time.Since(start))
		if done, err := condition(&pod); done {
			if err == nil {
				DefaultLogger.Infof("Pod %q satisfied condition %q", podName, desc)
			}
			return err
		}
	}
	return fmt.Errorf("Gave up after waiting %v for pod %q to be %q", timeout, podName, desc)
}

type podCondition func(pod *corev1.Pod) (bool, error)

// checkPodsCondition returns whether all pods whose names are listed in podNames
// in namespace ns are in the condition, using c and waiting at most timeout.
func checkPodsCondition(c ctrlruntimeclient.Client, ns string, podNames []string, timeout time.Duration, condition podCondition, desc string) bool {
	np := len(podNames)
	DefaultLogger.Infof("Waiting up to %v for %d pods to be %s: %s", timeout, np, desc, podNames)
	type waitPodResult struct {
		success bool
		podName string
	}
	result := make(chan waitPodResult, len(podNames))
	for _, podName := range podNames {
		// Launch off pod readiness checkers.
		go func(name string) {
			err := WaitForPodCondition(c, ns, name, desc, timeout, condition)
			result <- waitPodResult{err == nil, name}
		}(podName)
	}
	// Wait for them all to finish.
	success := true
	for range podNames {
		res := <-result
		if !res.success {
			DefaultLogger.Debugf("Pod %[1]s failed to be %[2]s.", res.podName, desc)
			success = false
		}
	}
	DefaultLogger.Infof("Wanted all %d pods to be %s. Result: %t. Pods: %v", np, desc, success, podNames)
	return success
}

// PodRunningReady checks whether pod p's phase is running and it has a ready
// condition of status true.
func PodRunningReady(p *corev1.Pod) (bool, error) {
	// Check the phase is running.
	if p.Status.Phase != corev1.PodRunning {
		return false, fmt.Errorf("want pod '%s' on '%s' to be '%v' but was '%v'",
			p.ObjectMeta.Name, p.Spec.NodeName, corev1.PodRunning, p.Status.Phase)
	}
	// Check the ready condition is true.
	if !podutils.IsPodReady(p) {
		return false, fmt.Errorf("pod '%s' on '%s' didn't have condition {%v %v}; conditions: %v",
			p.ObjectMeta.Name, p.Spec.NodeName, corev1.PodReady, corev1.ConditionTrue, p.Status.Conditions)
	}
	return true, nil
}
