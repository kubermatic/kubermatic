package clusterdeletion

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/machine-controller/pkg/node/eviction"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	utilpointer "k8s.io/utils/pointer"
	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	controllerruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	deletedLBAnnotationName = "kubermatic.io/cleaned-up-loadbalancers"
)

func New(seedClient, userClusterClient controllerruntimeclient.Client) *Deletion {
	return &Deletion{
		seedClient:        seedClient,
		userClusterClient: userClusterClient,
	}
}

type Deletion struct {
	seedClient        controllerruntimeclient.Client
	userClusterClient controllerruntimeclient.Client
}

// cleanupCluster is responsible for cleaning up a cluster
func (d *Deletion) CleanupCluster(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	err := d.cleanupCluster(ctx, cluster)
	result := &reconcile.Result{}
	if cluster != nil && len(cluster.Finalizers) > 0 {
		result.RequeueAfter = 10 * time.Second
	}

	return result, err
}

func (d *Deletion) cleanupCluster(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	// Delete Volumes and LB's inside the user cluster
	if err := d.cleanupInClusterResources(ctx, cluster); err != nil {
		return err
	}
	if err := d.deletingNodeCleanup(ctx, cluster); err != nil {
		return err
	}

	// If we still have nodes, we must not cleanup other infrastructure at the cloud provider
	if kuberneteshelper.HasFinalizer(cluster, kubermaticapiv1.NodeDeletionFinalizer) {
		return nil
	}

	return nil
}

func (d *Deletion) cleanupInClusterResources(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	shouldDeleteLBs := kuberneteshelper.HasFinalizer(cluster, kubermaticapiv1.InClusterLBCleanupFinalizer)
	shouldDeletePVs := kuberneteshelper.HasFinalizer(cluster, kubermaticapiv1.InClusterPVCleanupFinalizer)

	// If no relevant finalizer exists, directly return
	if !shouldDeleteLBs && !shouldDeletePVs {
		return nil
	}

	// We'll set this to true in case we deleted something. This is meant to requeue as long as all resources are really gone
	// We'll use it for LB's and PV's as well, so the Kubernetes controller manager does the cleanup of all resources in parallel
	var deletedSomeResource bool

	if shouldDeleteLBs {
		serviceList := &corev1.ServiceList{}
		if err := d.userClusterClient.List(ctx, &controllerruntimeclient.ListOptions{}, serviceList); err != nil {

			return fmt.Errorf("failed to list Service's from user cluster: %v", err)
		}

		for _, service := range serviceList.Items {
			// Need to change the scope so the inline func in the updateCluster call always has the service from the current iteration
			service := service
			if service.Spec.Type == corev1.ServiceTypeLoadBalancer {
				if err := d.userClusterClient.Delete(ctx, &service); err != nil {
					return fmt.Errorf("failed to delete Service '%s/%s' from user cluster: %v", service.Namespace, service.Name, err)
				}
				deletedSomeResource = true
				err := d.updateCluster(ctx, cluster, func(cluster *kubermaticv1.Cluster) {
					if cluster.Annotations == nil {
						cluster.Annotations = map[string]string{}
					}
					cluster.Annotations[deletedLBAnnotationName] = cluster.Annotations[deletedLBAnnotationName] + fmt.Sprintf(",%s", string(service.UID))
				})
				if err != nil {
					return fmt.Errorf("failed to update cluster when trying to add UID of deleted LoadBalancer: %v", err)
				}
				// Wait for the update to appear in the lister as we use the data from the lister later on to verify if the LoadBalancers
				// are gone
				if err := wait.Poll(10*time.Millisecond, 5*time.Second, func() (bool, error) {
					latestCluster := &kubermaticv1.Cluster{}
					if err := d.seedClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, latestCluster); err != nil {
						return false, err
					}
					if strings.Contains(latestCluster.Annotations[deletedLBAnnotationName], string(service.UID)) {
						return true, nil
					}
					return false, nil
				}); err != nil {
					return fmt.Errorf("failed to wait for deletedLBAnnotation to appear in the lister: %v", err)
				}
			}
		}
	}

	if shouldDeletePVs {
		// Prevent re-creation of PVs, PVCs and PDBs by using an intentionally defunct admissionWebhook
		admissionWebhookName, admissionWebhook := terminatingAdmissionWebhook()
		err := d.userClusterClient.Get(ctx, types.NamespacedName{Name: admissionWebhookName}, &admissionregistrationv1beta1.ValidatingWebhookConfiguration{})
		if err != nil && !kerrors.IsNotFound(err) {
			return fmt.Errorf("error checking if %q webhook configuration already exists: %v", admissionWebhookName, err)
		}
		if kerrors.IsNotFound(err) {
			if err := d.userClusterClient.Create(ctx, admissionWebhook); err != nil {
				return fmt.Errorf("failed to create %q webhook configuration: %v", admissionWebhookName, err)
			}
		}

		pdbs := &policyv1beta1.PodDisruptionBudgetList{}
		if err := d.userClusterClient.List(ctx, &controllerruntimeclient.ListOptions{}, pdbs); err != nil {
			return fmt.Errorf("failed to list pdbs: %v", err)
		}
		for _, pdb := range pdbs.Items {
			if err := d.userClusterClient.Delete(ctx, &pdb); err != nil {
				return fmt.Errorf("failed to delete pdb '%s/%s': %v", pdb.Namespace, pdb.Name, err)
			}
		}
		// Make sure we don't continue until all PDBs are actually gone
		if len(pdbs.Items) > 0 {
			return nil
		}
		// Delete all workloads that use PVs. We must do this before we clean up the node, otherwise
		// we end up in a deadlock when CSI is used
		cleanedSomethingUp, err := d.cleanupPVUsingWorkloads(ctx)
		if err != nil {
			return fmt.Errorf("failed to clean up PV using workloads from user cluster: %v", err)
		}
		if cleanedSomethingUp {
			return nil
		}

		// Delete PVC's
		pvcList := &corev1.PersistentVolumeClaimList{}
		if err := d.userClusterClient.List(ctx, &controllerruntimeclient.ListOptions{}, pvcList); err != nil {
			return fmt.Errorf("failed to list PVCs from user cluster: %v", err)
		}

		for _, pvc := range pvcList.Items {
			if err := d.userClusterClient.Delete(ctx, &pvc); err != nil {
				return fmt.Errorf("failed to delete PVC '%s/%s' from user cluster: %v", pvc.Namespace, pvc.Name, err)
			}
			deletedSomeResource = true
		}

		// Delete PV's
		pvList := &corev1.PersistentVolumeList{}
		if err := d.userClusterClient.List(ctx, &controllerruntimeclient.ListOptions{}, pvList); err != nil {
			return fmt.Errorf("failed to list PVs from user cluster: %v", err)
		}

		for _, pv := range pvList.Items {
			if err := d.userClusterClient.Delete(ctx, &pv); err != nil {
				return fmt.Errorf("failed to delete PV '%s' from user cluster: %v", pv.Name, err)
			}
			deletedSomeResource = true
		}
	}

	// If we deleted something it is implied that there was still something left. Just return
	// here so the finalizers stay, it will make the cluster controller requeue us after a delay
	// This also means that we may end up issuing multiple DELETE calls against the same ressource
	// if cleaning up takes some time, but that shouldn't cause any harm
	// We also need to return when something was deleted so the checkIfAllLoadbalancersAreGone
	// call gets an updated version of the cluster from the lister
	if deletedSomeResource {
		return nil
	}

	lbsAreGone, err := d.checkIfAllLoadbalancersAreGone(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to check if all Loadbalancers are gone: %v", err)
	}
	// Return so we check again later
	if !lbsAreGone {
		return nil
	}

	return d.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
		kuberneteshelper.RemoveFinalizer(c, kubermaticapiv1.InClusterLBCleanupFinalizer)
		kuberneteshelper.RemoveFinalizer(c, kubermaticapiv1.InClusterPVCleanupFinalizer)
	})
}

func (d *Deletion) deletingNodeCleanup(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if !kuberneteshelper.HasFinalizer(cluster, kubermaticapiv1.NodeDeletionFinalizer) {
		return nil
	}

	nodes := &corev1.NodeList{}
	if err := d.userClusterClient.List(ctx, &controllerruntimeclient.ListOptions{}, nodes); err != nil {
		return fmt.Errorf("failed to get user cluster nodes: %v", err)
	}

	// If we delete a cluster, we should disable the eviction on the nodes
	for _, node := range nodes.Items {
		if node.Annotations[eviction.SkipEvictionAnnotationKey] == "true" {
			continue
		}

		err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			// Get latest version of the node to prevent conflict errors
			currentNode := &corev1.Node{}
			if err := d.userClusterClient.Get(ctx, types.NamespacedName{Name: node.Name}, currentNode); err != nil {
				return err
			}
			if currentNode.Annotations == nil {
				currentNode.Annotations = map[string]string{}
			}
			currentNode.Annotations[eviction.SkipEvictionAnnotationKey] = "true"

			return d.userClusterClient.Update(ctx, currentNode)
		})
		if err != nil {
			return fmt.Errorf("failed to add the annotation '%s=true' to node '%s': %v", eviction.SkipEvictionAnnotationKey, node.Name, err)
		}
	}

	machineDeploymentList := &clusterv1alpha1.MachineDeploymentList{}
	listOpts := &controllerruntimeclient.ListOptions{Namespace: metav1.NamespaceSystem}
	if err := d.userClusterClient.List(ctx, listOpts, machineDeploymentList); err != nil {
		return fmt.Errorf("failed to list MachineDeployments: %v", err)
	}
	if len(machineDeploymentList.Items) > 0 {
		// TODO: Use DeleteCollection once https://github.com/kubernetes-sigs/controller-runtime/issues/344 is resolved
		for _, machineDeployment := range machineDeploymentList.Items {
			if err := d.userClusterClient.Delete(ctx, &machineDeployment); err != nil {
				return fmt.Errorf("failed to delete MachineDeployment %q: %v", machineDeployment.Name, err)
			}
		}
		// Return here to make sure we don't attempt to delete MachineSets until the MachineDeployment is actually gone
		return nil
	}

	machineSetList := &clusterv1alpha1.MachineSetList{}
	if err := d.userClusterClient.List(ctx, listOpts, machineSetList); err != nil {
		return fmt.Errorf("failed to list MachineSets: %v", err)
	}
	if len(machineSetList.Items) > 0 {
		// TODO: Use DeleteCollection once https://github.com/kubernetes-sigs/controller-runtime/issues/344 is resolved
		for _, machineSet := range machineSetList.Items {
			if err := d.userClusterClient.Delete(ctx, &machineSet); err != nil {
				return fmt.Errorf("failed to delete MachineSet %q: %v", machineSet.Name, err)
			}
		}
		// Return here to make sure we don't attempt to delete Machines until the MachineSet is actually gone
		return nil
	}

	machineList := &clusterv1alpha1.MachineList{}
	if err := d.userClusterClient.List(ctx, listOpts, machineList); err != nil {
		return fmt.Errorf("failed to get Machines: %v", err)
	}
	if len(machineList.Items) > 0 {
		// TODO: Use DeleteCollection once https://github.com/kubernetes-sigs/controller-runtime/issues/344 is resolved
		for _, machine := range machineList.Items {
			if err := d.userClusterClient.Delete(ctx, &machine); err != nil {
				return fmt.Errorf("failed to delete Machine %q: %v", machine.Name, err)
			}
		}

		return nil
	}

	return d.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
		kuberneteshelper.RemoveFinalizer(c, kubermaticapiv1.NodeDeletionFinalizer)
	})
}

// checkIfAllLoadbalancersAreGone checks if all the services of type LoadBalancer were successfully
// deleted. The in-tree cloud providers do this without a finalizer and only after the service
// object is gone from the API, the only way to check is to wait for the relevant event
func (d *Deletion) checkIfAllLoadbalancersAreGone(ctx context.Context, cluster *kubermaticv1.Cluster) (bool, error) {
	// This check is only required for in-tree cloud provider that support LoadBalancers
	// TODO once we start external cloud controllers for one of these three: Make this check
	// a bit smarter, external cloud controllers will most likely not emit the event we wait for
	if cluster.Spec.Cloud.AWS == nil && cluster.Spec.Cloud.Azure == nil && cluster.Spec.Cloud.Openstack == nil {
		return true, nil
	}

	// We only need to wait for this if there were actually services of type Loadbalancer deleted
	if cluster.Annotations[deletedLBAnnotationName] == "" {
		return true, nil
	}

	deletedLoadBalancers := sets.NewString(strings.Split(strings.TrimPrefix(cluster.Annotations[deletedLBAnnotationName], ","), ",")...)

	// Kubernetes gives no guarantees at all about events, it is possible we don't get the event
	// so bail out after 2h
	if cluster.DeletionTimestamp.UTC().Add(2 * time.Hour).Before(time.Now().UTC()) {
		staleLBs.WithLabelValues(cluster.Name).Set(float64(deletedLoadBalancers.Len()))
		return true, nil
	}

	for deletedLB := range deletedLoadBalancers {
		selector := fields.OneTermEqualSelector("involvedObject.uid", deletedLB)
		events := &corev1.EventList{}
		if err := d.userClusterClient.List(context.Background(), &controllerruntimeclient.ListOptions{FieldSelector: selector}, events); err != nil {
			return false, fmt.Errorf("failed to get service events: %v", err)
		}
		for _, event := range events.Items {
			if event.Reason == "DeletedLoadBalancer" {
				deletedLoadBalancers.Delete(deletedLB)
			}
		}

	}
	err := d.updateCluster(ctx, cluster, func(cluster *kubermaticv1.Cluster) {
		if deletedLoadBalancers.Len() > 0 {
			cluster.Annotations[deletedLBAnnotationName] = strings.Join(deletedLoadBalancers.List(), ",")
		} else {
			delete(cluster.Annotations, deletedLBAnnotationName)
		}
	})
	if err != nil {
		return false, fmt.Errorf("failed to update cluster: %v", err)
	}
	if deletedLoadBalancers.Len() > 0 {
		return false, nil
	}
	return true, nil
}

func (d *Deletion) updateCluster(ctx context.Context, cluster *kubermaticv1.Cluster, modify func(*kubermaticv1.Cluster)) error {
	// Store it here because it may be unset later on if an update request failed
	name := cluster.Name
	return retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		//Get latest version
		if err := d.seedClient.Get(ctx, types.NamespacedName{Name: name}, cluster); err != nil {
			return err
		}
		// Apply modifications
		modify(cluster)
		// Update the cluster
		return d.seedClient.Update(ctx, cluster)
	})
}

func (d *Deletion) cleanupPVUsingWorkloads(ctx context.Context) (bool, error) {
	podList := &corev1.PodList{}
	if err := d.userClusterClient.List(ctx, &controllerruntimeclient.ListOptions{}, podList); err != nil {
		return false, fmt.Errorf("failed to list Pods from user cluster: %v", err)
	}

	pvUsingPods := []*corev1.Pod{}
	for _, pod := range podList.Items {
		if podUsesPV(&pod) {
			pvUsingPods = append(pvUsingPods, &pod)
		}
	}

	hasPVUsingPods := len(pvUsingPods) > 0

	wg := &sync.WaitGroup{}
	wg.Add(len(pvUsingPods))
	errs := []error{}
	errLock := &sync.Mutex{}
	for _, pod := range pvUsingPods {
		go func(p *corev1.Pod) {
			if err := d.resolveAndDeleteTopLevelUserClusterOwner(ctx, "Pod", p.Namespace, p.Name); err != nil {
				errLock.Lock()
				defer errLock.Unlock()
				errs = append(errs, err)
			}
			wg.Done()
		}(pod)
	}
	wg.Wait()

	if len(errs) > 0 {
		return hasPVUsingPods, fmt.Errorf("%v", errs)
	}

	return hasPVUsingPods, nil
}

func (d *Deletion) resolveAndDeleteTopLevelUserClusterOwner(ctx context.Context, kind, namespace, name string) error {
	object, err := getObjectForKind(kind)
	if err != nil {
		return err
	}
	if err := d.userClusterClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, object); err != nil {
		// With background deletion the owners will be gone before the dependants are gone
		if kerrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get object %q/%q of kind %q: %v", namespace, name, kind, err)
	}
	metav1Object, ok := object.(metav1.Object)
	if !ok {
		return fmt.Errorf("failed to assert object %q/%q of kind %q as metav1.Object", namespace, name, kind)
	}
	for _, ownerRef := range metav1Object.GetOwnerReferences() {
		if err := d.resolveAndDeleteTopLevelUserClusterOwner(ctx, ownerRef.Kind, metav1Object.GetNamespace(), ownerRef.Name); err != nil {
			return err
		}
	}

	return d.userClusterClient.Delete(ctx, object)
}

func podUsesPV(p *corev1.Pod) bool {
	for _, volume := range p.Spec.Volumes {
		if volume.VolumeSource.PersistentVolumeClaim != nil {
			return true
		}
	}
	return false
}

func getObjectForKind(kind string) (runtime.Object, error) {
	switch kind {
	case "DaemonSet":
		return &appsv1.DaemonSet{}, nil
	case "StatefulSet":
		return &appsv1.StatefulSet{}, nil
	case "ReplicaSet":
		return &appsv1.ReplicaSet{}, nil
	case "Deployment":
		return &appsv1.Deployment{}, nil
	case "Pod":
		return &corev1.Pod{}, nil
	default:
		return nil, fmt.Errorf("kind %q is unknown", kind)
	}
}

func terminatingAdmissionWebhook() (string, *admissionregistrationv1beta1.ValidatingWebhookConfiguration) {
	name := "kubernetes-cluster-cleanup"
	failurePolicy := admissionregistrationv1beta1.Fail
	return name, &admissionregistrationv1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				"description": "This webhok configuration exists to prevent creation of any new stateful resources in a cluster that is currently being terminated",
			},
		},
		Webhooks: []admissionregistrationv1beta1.Webhook{
			{
				// Must be a domain with at least three segments separated by dots
				Name: "kubernetes.cluster.cleanup",
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
					URL: utilpointer.StringPtr("https://127.0.0.1:1"),
				},
				Rules: []admissionregistrationv1beta1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1beta1.OperationType{admissionregistrationv1beta1.Create},
						Rule: admissionregistrationv1beta1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"*"},
							Resources:   []string{"persistentvolumes", "persistentvolumeclaims"},
						},
					},
					{
						Operations: []admissionregistrationv1beta1.OperationType{admissionregistrationv1beta1.Create},
						Rule: admissionregistrationv1beta1.Rule{
							APIGroups:   []string{"policy"},
							APIVersions: []string{"*"},
							Resources:   []string{"poddisruptionbudgets"},
						},
					},
				},
				FailurePolicy: &failurePolicy,
			},
		},
	}
}
