package usercluster

import (
	"context"
	"fmt"

	"github.com/golang/glog"

	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/machine-controller"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/metrics-server"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"

	controllerclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconcile creates, updates, or deletes Kubernetes resources to match the desired state.
func (r *reconciler) reconcile(ctx context.Context) error {
	if err := r.ensureAPIServices(ctx); err != nil {
		return err
	}

	if err := r.ensureRoles(ctx); err != nil {
		return err
	}
	return nil
}

// GetAPIServiceCreators returns a list of APIServiceCreator
func (r *reconciler) GetAPIServiceCreators() []resources.APIServiceCreator {
	creators := []resources.APIServiceCreator{}
	if !r.openshift {
		creators = append(creators, metricsserver.APIService)
	}
	return creators
}

func (r *reconciler) ensureAPIServices(ctx context.Context) error {
	creators := r.GetAPIServiceCreators()

	for _, create := range creators {
		apiService, err := create(nil)
		if err != nil {
			return fmt.Errorf("failed to build APIService: %v", err)
		}

		existing := &apiregistrationv1beta1.APIService{}
		err = r.Get(ctx, controllerclient.ObjectKey{Namespace: apiService.Namespace, Name: apiService.Name}, existing)
		if err != nil {
			if errors.IsNotFound(err) {
				if err := r.Create(ctx, apiService); err != nil {
					return fmt.Errorf("failed to create APIService %s in namespace %s: %v", apiService.Name, apiService.Namespace, err)
				}
				glog.V(2).Infof("Created a new APIService %s in namespace %s", apiService.Name, apiService.Namespace)
				return nil
			}
			return err
		}

		apiService, err = create(existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build APIService : %v", err)
		}
		if equality.Semantic.DeepEqual(apiService, existing) {
			continue
		}

		if err := r.Update(ctx, apiService); err != nil {
			return fmt.Errorf("failed to update APIService %s in namespace %s: %v", apiService.Name, apiService.Namespace, err)
		}
		glog.V(4).Infof("Updated the APIService %s in namespace %s", apiService.Name, apiService.Namespace)
	}

	return nil
}

func (r *reconciler) ensureRoles(ctx context.Context) error {
	// kube-system
	creators := []resources.NamedRoleCreatorGetter{
		machinecontroller.KubeSystemRoleCreator(),
	}

	if err := resources.ReconcileRoles(creators, v1.NamespaceSystem, r.Client, r.cache); err != nil {
		return fmt.Errorf("failed to reconcile Roles in the namespace %s: %v", v1.NamespaceSystem, err)
	}

	// kube-public
	creators = []resources.NamedRoleCreatorGetter{
		machinecontroller.ClusterInfoReaderRoleCreator(),
		machinecontroller.KubePublicRoleCreator(),
	}

	if err := resources.ReconcileRoles(creators, v1.NamespacePublic, r.Client, r.cache); err != nil {
		return fmt.Errorf("failed to reconcile Roles in the namespace %s: %v", v1.NamespacePublic, err)
	}

	// default
	creators = []resources.NamedRoleCreatorGetter{
		machinecontroller.EndpointReaderRoleCreator(),
	}

	if err := resources.ReconcileRoles(creators, v1.NamespaceDefault, r.Client, r.cache); err != nil {
		return fmt.Errorf("failed to reconcile Roles in the namespace %s: %v", v1.NamespaceDefault, err)
	}

	return nil
}
