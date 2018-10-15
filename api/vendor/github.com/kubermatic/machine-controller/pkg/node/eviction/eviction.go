package eviction

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/util/retry"
)

const (
	timeout                   = 60 * time.Second
	SkipEvictionAnnotationKey = "kubermatic.io/skip-eviction"
)

type NodeEviction struct {
	nodeName   string
	nodeLister listerscorev1.NodeLister
	client     kubernetes.Interface
}

// New returns a new NodeEviction
func New(nodeName string, nodeLister listerscorev1.NodeLister, client kubernetes.Interface) *NodeEviction {
	return &NodeEviction{
		nodeName:   nodeName,
		nodeLister: nodeLister,
		client:     client,
	}
}

// Run excutes the eviction
func (ne *NodeEviction) Run() error {
	listerNode, err := ne.nodeLister.Get(ne.nodeName)
	if err != nil {
		return fmt.Errorf("failed to get node from lister: %v", err)
	}
	node := listerNode.DeepCopy()
	if _, exists := node.Annotations[SkipEvictionAnnotationKey]; exists {
		glog.V(4).Infof("Skipping eviction for node %s as it has a %s annotation", ne.nodeName, SkipEvictionAnnotationKey)
		return nil
	}
	glog.V(4).Infof("Starting to evict node %s", ne.nodeName)

	if err := ne.cordonNode(node); err != nil {
		return fmt.Errorf("failed to cordon node %s: %v", ne.nodeName, err)
	}
	glog.V(6).Infof("Successfully cordoned node %s", ne.nodeName)

	podsToEvict, err := ne.getFilteredPods()
	if err != nil {
		return fmt.Errorf("failed to get Pods to evict for node %s: %v", ne.nodeName, err)
	}
	glog.V(6).Infof("Found %v pods to evict for node %s", len(podsToEvict), ne.nodeName)

	if errs := ne.evictPods(podsToEvict); len(errs) > 0 {
		return fmt.Errorf("failed to evict pods, errors encountered: %v", errs)
	}
	glog.V(6).Infof("Successfully created evictions for all pods on node %s!", ne.nodeName)

	glog.V(6).Infof("Waiting for deletion of all pods for node %s", ne.nodeName)
	if err := ne.waitForDeletion(podsToEvict); err != nil {
		return fmt.Errorf("failed waiting for pods of node %s to be deleted: %v", ne.nodeName, err)
	}
	glog.V(4).Infof("All pods of node %s were successfully evicted", ne.nodeName)

	return nil
}

func (ne *NodeEviction) cordonNode(node *corev1.Node) error {
	_, err := ne.updateNode(func(n *corev1.Node) {
		n.Spec.Unschedulable = true
	})
	if err != nil {
		return err
	}

	// Be paranoid and wait until the change got propagated to the lister
	// This assumes that the delay between our lister and the APIserver
	// is smaller or equal to the delay the schedulers lister has - If
	// that is not the case, there is a small chance the scheduler schedules
	// pods in between, those will then get deleted upon node deletion and
	// not evicted
	return wait.Poll(1*time.Second, timeout, func() (bool, error) {
		node, err := ne.nodeLister.Get(ne.nodeName)
		if err != nil {
			return false, err
		}
		if node.Spec.Unschedulable {
			return true, nil
		}
		return false, nil
	})
}

func (ne *NodeEviction) getFilteredPods() ([]corev1.Pod, error) {
	pods, err := ne.client.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": ne.nodeName}).String(),
	})
	if err != nil {
		return nil, err
	}

	var filteredPods []corev1.Pod
	for _, candidatePod := range pods.Items {
		if candidatePod.Status.Phase == corev1.PodSucceeded || candidatePod.Status.Phase == corev1.PodFailed {
			continue
		}
		if controllerRef := metav1.GetControllerOf(&candidatePod); controllerRef != nil && controllerRef.Kind == "DaemonSet" {
			continue
		}
		if _, found := candidatePod.ObjectMeta.Annotations[corev1.MirrorPodAnnotationKey]; found {
			continue
		}
		filteredPods = append(filteredPods, candidatePod)
	}

	return filteredPods, nil
}

func (ne *NodeEviction) evictPods(pods []corev1.Pod) []error {
	if len(pods) == 0 {
		return nil
	}

	errCh := make(chan error, len(pods))
	retErrs := []error{}

	var wg sync.WaitGroup
	var isDone bool
	defer func() { isDone = true }()

	wg.Add(len(pods))
	for _, pod := range pods {
		go func(p corev1.Pod) {
			defer wg.Done()
			for {
				if isDone {
					return
				}
				err := ne.evictPod(&p)
				if err == nil || kerrors.IsNotFound(err) {
					glog.V(6).Infof("Successfully evicted pod %s/%s on node %s", p.Namespace, p.Name, ne.nodeName)
					return
				} else if kerrors.IsTooManyRequests(err) {
					glog.V(6).Infof("Will retry eviction for pod %s/%s on node %s", p.Namespace, p.Name, ne.nodeName)
					time.Sleep(5 * time.Second)
				} else {
					errCh <- fmt.Errorf("error evicting pod %s/%s on node %s: %v", p.Namespace, p.Name, ne.nodeName, err)
					return
				}
			}
		}(pod)
	}

	finished := make(chan struct{})
	go func() { wg.Wait(); finished <- struct{}{} }()

	select {
	case <-finished:
		glog.V(6).Infof("All goroutines for eviction pods on node %s finished", ne.nodeName)
		break
	case err := <-errCh:
		glog.V(6).Infof("Got an error from eviction goroutine for node %s: %v", ne.nodeName, err)
		retErrs = append(retErrs, err)
	case <-time.After(timeout):
		retErrs = append(retErrs, fmt.Errorf("timed out waiting for evictions to complete"))
		glog.V(6).Infof("Timed out waiting for all evition goroutiness for node %s to finish", ne.nodeName)
		break
	}

	return retErrs
}

func (ne *NodeEviction) evictPod(pod *corev1.Pod) error {
	eviction := &policy.Eviction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
	}
	return ne.client.PolicyV1beta1().Evictions(eviction.Namespace).Evict(eviction)
}

func (ne *NodeEviction) updateNode(modify func(*corev1.Node)) (*corev1.Node, error) {
	var updatedNode *corev1.Node
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var retryErr error

		//Get latest version from API
		currentNode, err := ne.client.CoreV1().Nodes().Get(ne.nodeName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		// Apply modifications
		modify(currentNode)
		// Update the node
		updatedNode, retryErr = ne.client.CoreV1().Nodes().Update(currentNode)
		return retryErr
	})

	return updatedNode, err
}

func (ne *NodeEviction) waitForDeletion(pods []corev1.Pod) error {
	return wait.Poll(1*time.Second, timeout, func() (bool, error) {
		for _, pod := range pods {
			_, err := ne.client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
			if err != nil && kerrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return true, nil
	})
}
