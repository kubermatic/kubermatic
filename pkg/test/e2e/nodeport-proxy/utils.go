package nodeport_proxy

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/util/podutils"

	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const (
	// poll is how often to poll pods, nodes and claims.
	poll = 2 * time.Second
)

var (
	logger = CreateLogger(false)
)

// CreateLogger creates a new Logger
func CreateLogger(debug bool) *zap.SugaredLogger {
	return ctrlzap.NewRaw(ctrlzap.UseDevMode(debug), ctrlzap.WriteTo(GinkgoWriter)).Sugar()
}

func GetClientsOrDie() (ctrlclient.Client, rest.Interface, *rest.Config) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	config := ctrl.GetConfigOrDie()
	mapper, err := apiutil.NewDynamicRESTMapper(config)
	if err != nil {
		panic(errors.Wrap(err, "failed to create dynamic REST mapper"))
	}
	gvk, err := apiutil.GVKForObject(&corev1.Pod{}, scheme)
	if err != nil {
		panic(errors.Wrap(err, "failed to get pod GVK"))
	}
	podRestClient, err := apiutil.RESTClientForGVK(gvk, config, serializer.NewCodecFactory(scheme))
	if err != nil {
		panic(errors.Wrap(err, "failed to create pod rest client"))
	}
	c, err := ctrlclient.New(config, ctrlclient.Options{
		Mapper: mapper,
		Scheme: scheme,
	})
	if err != nil {
		panic(err)
	}
	return c, podRestClient, config
}

// function used to extract port info
type extractPortFunc func(corev1.ServicePort) int32

func extractPort(svc *corev1.Service, extract extractPortFunc) sets.Int32 {
	res := sets.NewInt32()
	for _, p := range svc.Spec.Ports {
		val := extract(p)
		if val != 0 {
			res.Insert(val)
		}
	}
	return res
}

// ExtractNodePorts returns the set of node ports extracted from the given
// Service.
func ExtractNodePorts(svc *corev1.Service) sets.Int32 {
	return extractPort(svc,
		func(p corev1.ServicePort) int32 { return p.NodePort })
}

// ExtractPorts returns the set of ports extracted from the given
// Service.
func ExtractPorts(svc *corev1.Service) sets.Int32 {
	return extractPort(svc,
		func(p corev1.ServicePort) int32 { return p.Port })
}

// FindExposingNodePort returns the node port associated to the given target
// port.
func FindExposingNodePort(svc *corev1.Service, targetPort int32) int32 {
	for _, p := range svc.Spec.Ports {
		if p.TargetPort.IntVal == targetPort {
			return p.NodePort
		}
	}
	return 0
}

// WaitForPodsCreated waits for the given replicas number of pods matching the
// given set of labels to be created, and returns the names of the matched
// pods.
func WaitForPodsCreated(c ctrlclient.Client, replicas int, namespace string, matchLabels map[string]string) ([]string, error) {
	timeout := 2 * time.Minute
	// List the pods, making sure we observe all the replicas.
	logger.Debugf("Waiting up to %v for %d pods to be created", timeout, replicas)
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(2 * time.Second) {
		pods := corev1.PodList{}
		if err := c.List(context.TODO(), &pods,
			ctrlclient.InNamespace(namespace),
			ctrlclient.MatchingLabels(matchLabels)); err != nil {
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
			logger.Infof("Found all %d pods", replicas)
			return found, nil
		}
		logger.Debugf("Found %d/%d pods - will retry", len(found), replicas)
	}
	return nil, fmt.Errorf("timeout waiting for %d pods to be created", replicas)
}

// CheckPodsRunningReady returns whether all pods whose names are listed in
// podNames in namespace ns are running and ready, using c and waiting at most
// timeout.
func CheckPodsRunningReady(c ctrlclient.Client, ns string, podNames []string, timeout time.Duration) bool {
	return checkPodsCondition(c, ns, podNames, timeout, PodRunningReady, "running and ready")
}

// WaitForPodCondition waits a pods to be matched to the given condition.
func WaitForPodCondition(c ctrlclient.Client, ns, podName, desc string, timeout time.Duration, condition podCondition) error {
	logger.Infof("Waiting up to %v for pod %q in namespace %q to be %q", timeout, podName, ns, desc)
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		pod := corev1.Pod{}
		if err := c.Get(context.TODO(), types.NamespacedName{Name: podName, Namespace: ns}, &pod); err != nil {
			if apierrors.IsNotFound(err) {
				logger.Debugf("Pod %q in namespace %q not found. Error: %v", podName, ns, err)
				return err
			}
			logger.Debugf("Get pod %q in namespace %q failed, ignoring for %v. Error: %v", podName, ns, poll, err)
			continue
		}
		// log now so that current pod info is reported before calling `condition()`
		logger.Debugf("Pod %q: Phase=%q, Reason=%q, readiness=%t. Elapsed: %v",
			podName, pod.Status.Phase, pod.Status.Reason, podutils.IsPodReady(&pod), time.Since(start))
		if done, err := condition(&pod); done {
			if err == nil {
				logger.Infof("Pod %q satisfied condition %q", podName, desc)
			}
			return err
		}
	}
	return fmt.Errorf("Gave up after waiting %v for pod %q to be %q", timeout, podName, desc)
}

type podCondition func(pod *corev1.Pod) (bool, error)

// checkPodsCondition returns whether all pods whose names are listed in podNames
// in namespace ns are in the condition, using c and waiting at most timeout.
func checkPodsCondition(c ctrlclient.Client, ns string, podNames []string, timeout time.Duration, condition podCondition, desc string) bool {
	np := len(podNames)
	logger.Infof("Waiting up to %v for %d pods to be %s: %s", timeout, np, desc, podNames)
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
			logger.Debugf("Pod %[1]s failed to be %[2]s.", res.podName, desc)
			success = false
		}
	}
	logger.Infof("Wanted all %d pods to be %s. Result: %t. Pods: %v", np, desc, success, podNames)
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
