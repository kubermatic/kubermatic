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

// Reconcile creates, updates, or deletes Kubernetes resources to match the desired state.
func (r *reconciler) reconcile(ctx context.Context) error {
	if err := r.ensureAPIServices(ctx); err != nil {
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

		if err := r.Update(ctx, apiService); err != nil {
			return fmt.Errorf("failed to update APIService %s in namespace %s: %v", apiService.Name, apiService.Namespace, err)
		}
		glog.V(debugLevel).Infof("updated APIService %s in namespace %s", apiService.Name, apiService.Namespace)
	}

	return nil
}
