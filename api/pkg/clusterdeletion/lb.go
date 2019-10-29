package clusterdeletion

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	controllerruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	eventReasonDeletedLoadBalancer = "DeletedLoadBalancer"
)

func (d *Deletion) cleanupLBs(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (deletedSomeLBs bool, err error) {
	log = log.Named("lb-cleanup")
	log.Debug("Cleaning up LoadBalancers...")

	userClusterClient, err := d.userClusterClientGetter()
	if err != nil {
		return false, err
	}

	serviceList := &corev1.ServiceList{}
	if err := userClusterClient.List(ctx, serviceList); err != nil {
		return false, fmt.Errorf("failed to list Service's from user cluster: %v", err)
	}

	for _, service := range serviceList.Items {
		serviceName := fmt.Sprintf("%s/%s", service.Namespace, service.Name)
		slog := log.With("service", serviceName)

		// Only LoadBalancer services create cost on cloud providers
		if service.Spec.Type != corev1.ServiceTypeLoadBalancer {
			slog.Debug("Skipping cleanup of service as it's not a LoadBalancer")
			continue
		}

		if err := d.cleanupLB(ctx, slog, userClusterClient, &service, cluster); err != nil {
			return deletedSomeLBs, fmt.Errorf("failed to delete service %q inside user cluster: %v", serviceName, err)
		}
		deletedSomeLBs = true
	}

	return deletedSomeLBs, nil
}

func (d *Deletion) cleanupLB(ctx context.Context, log *zap.SugaredLogger, userClusterClient controllerruntimeclient.Client, service *corev1.Service, cluster *kubermaticv1.Cluster) error {
	log.Debug("Deleting service...")
	if err := userClusterClient.Delete(ctx, service); err != nil {
		return fmt.Errorf("failed to delete service: %v", err)
	}
	log.Info("Deleted service")

	// We store the deleted service UID's on the cluster so we can check on the next iteration if they are really gone.
	// We only really know if a LoadBalancer(The cloud provider LB) is gone, until there has been an event stating that.
	// The event contains the UID of the corresponding, deleted, service.
	// Upstream changed that recently to use a finalizer - As soon as we only support Kubernetes versions above that, we can remove this
	oldCluster := cluster.DeepCopy()
	if cluster.Annotations == nil {
		cluster.Annotations = map[string]string{}
	}
	cluster.Annotations[deletedLBAnnotationName] += fmt.Sprintf(",%s", string(service.UID))
	if err := d.seedClient.Patch(ctx, cluster, controllerruntimeclient.MergeFrom(oldCluster)); err != nil {
		return fmt.Errorf("failed to update cluster when trying to add UID of deleted LoadBalancer: %v", err)
	}

	// Wait for the update to appear in the lister as we use the data from the lister later on to verify if the LoadBalancers
	// are gone
	if err := wait.Poll(10*time.Millisecond, 5*time.Second, func() (bool, error) {
		latestCluster := &kubermaticv1.Cluster{}
		if err := d.seedClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, latestCluster); err != nil {
			return false, err
		}
		return strings.Contains(latestCluster.Annotations[deletedLBAnnotationName], string(service.UID)), nil
	}); err != nil {
		return fmt.Errorf("failed to wait for deletedLBAnnotation to appear in the lister: %v", err)
	}

	return nil
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

	userClusterClient, err := d.userClusterClientGetter()
	if err != nil {
		return false, err
	}
	for deletedLB := range deletedLoadBalancers {
		selector := fields.OneTermEqualSelector("involvedObject.uid", deletedLB)
		events := &corev1.EventList{}
		if err := userClusterClient.List(context.Background(), events, &controllerruntimeclient.ListOptions{FieldSelector: selector}); err != nil {
			return false, fmt.Errorf("failed to get service events: %v", err)
		}
		for _, event := range events.Items {
			if event.Reason == eventReasonDeletedLoadBalancer {
				deletedLoadBalancers.Delete(deletedLB)
			}
		}
	}

	oldCluster := cluster.DeepCopy()
	cluster.Annotations[deletedLBAnnotationName] = strings.Join(deletedLoadBalancers.List(), ",")
	if err := d.seedClient.Patch(ctx, cluster, controllerruntimeclient.MergeFrom(oldCluster)); err != nil {
		return false, fmt.Errorf("failed to update cluster: %v", err)
	}

	return deletedLoadBalancers.Len() > 0, nil
}
