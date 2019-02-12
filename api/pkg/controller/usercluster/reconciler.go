package usercluster

import (
	"context"
	"fmt"

	"github.com/golang/glog"

	controllerclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/metrics-server"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
)

const (
	debugLevel = 4
)

// reconciler creates and deletes Kubernetes resources to achieve the desired state
type reconciler struct {
	ctx    context.Context
	client controllerclient.Client
}

// Reconcile creates, updates, or deletes Kubernetes resources to match the desired state.
func (r *reconciler) Reconcile() error {
	if err := r.ensureAPIServices(); err != nil {
		return err
	}
	return nil
}

// GetAPIServiceCreators returns a list of APIServiceCreator
func GetAPIServiceCreators() []resources.APIServiceCreator {
	return []resources.APIServiceCreator{
		metricsserver.APIService,
	}
}

func (r *reconciler) ensureAPIServices() error {
	creators := GetAPIServiceCreators()

	for _, create := range creators {
		apiService, err := create(nil)
		if err != nil {
			return fmt.Errorf("failed to build APIService: %v", err)
		}

		existing := &apiregistrationv1beta1.APIService{}
		err = r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: apiService.Namespace, Name: apiService.Name}, existing)
		if err != nil {
			if errors.IsNotFound(err) {
				if err := r.client.Create(r.ctx, apiService); err != nil {
					return fmt.Errorf("failed to create APIService %s in namespace %s: %v", apiService.Name, apiService.Namespace, err)
				}
				glog.V(debugLevel).Infof("created a new APIService %s in namespace %s", apiService.Name, apiService.Namespace)
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

		if err := r.client.Update(r.ctx, apiService); err != nil {
			return fmt.Errorf("failed to update APIService %s in namespace %s: %v", apiService.Name, apiService.Namespace, err)
		}
		glog.V(debugLevel).Infof("updated APIService %s in namespace %s", apiService.Name, apiService.Namespace)
	}

	return nil
}
