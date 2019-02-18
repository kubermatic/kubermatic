package usercluster

import (
	"context"
	"fmt"

	"github.com/golang/glog"

	controllerclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/controllermanager"
	"github.com/kubermatic/kubermatic/api/pkg/resources/ipamcontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machinecontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/metrics-server"
	"github.com/kubermatic/kubermatic/api/pkg/resources/scheduler"
	"github.com/kubermatic/kubermatic/api/pkg/resources/vpnsidecar"

	rbacv1 "k8s.io/api/rbac/v1"
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
	if err := r.ensureRoles(); err != nil {
		return err
	}
	if err := r.ensureRoleBindings(); err != nil {
		return err
	}
	if err := r.ensureClusterEnsureRoles(); err != nil {
		return err
	}
	if err := r.ensureClusterEnsureRoleBindings(); err != nil {
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

func (r *reconciler) ensureRoles() error {
	creators := []resources.RoleCreator{
		machinecontroller.EndpointReaderRole,
		machinecontroller.KubeSystemRole,
		machinecontroller.KubePublicRole,
		machinecontroller.ClusterInfoReaderRole,
	}

	for _, create := range creators {
		role, err := create(nil, nil)
		if err != nil {
			return fmt.Errorf("failed to build Role: %v", err)
		}

		existing := &rbacv1.Role{}
		err = r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: role.Namespace, Name: role.Name}, existing)
		if err != nil {
			if errors.IsNotFound(err) {
				if err := r.client.Create(r.ctx, role); err != nil {
					return fmt.Errorf("failed to create Role %s in namespace %s: %v", role.Name, role.Namespace, err)
				}
				glog.V(debugLevel).Infof("created a new Role %s in namespace %s", role.Name, role.Namespace)
				return nil
			}
			return err
		}

		role, err = create(nil, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build Role : %v", err)
		}
		if equality.Semantic.DeepEqual(role, existing) {
			continue
		}

		if err := r.client.Update(r.ctx, role); err != nil {
			return fmt.Errorf("failed to update Role %s in namespace %s: %v", role.Name, role.Namespace, err)
		}
		glog.V(debugLevel).Infof("updated Role %s in namespace %s", role.Name, role.Namespace)
	}

	return nil
}

func (r *reconciler) ensureRoleBindings() error {
	creators := []resources.RoleBindingCreator{
		machinecontroller.DefaultRoleBinding,
		machinecontroller.KubeSystemRoleBinding,
		machinecontroller.KubePublicRoleBinding,
		machinecontroller.ClusterInfoAnonymousRoleBinding,
		metricsserver.RolebindingAuthReader,
		scheduler.RoleBindingAuthDelegator,
		controllermanager.RoleBindingAuthDelegator,
	}

	for _, create := range creators {
		rb, err := create(nil, nil)
		if err != nil {
			return fmt.Errorf("failed to build RoleBinding: %v", err)
		}

		existing := &rbacv1.RoleBinding{}
		err = r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: rb.Namespace, Name: rb.Name}, existing)
		if err != nil {
			if errors.IsNotFound(err) {
				if err := r.client.Create(r.ctx, rb); err != nil {
					return fmt.Errorf("failed to create RoleBinding %s in namespace %s: %v", rb.Name, rb.Namespace, err)
				}
				glog.V(debugLevel).Infof("created a new RoleBinding %s in namespace %s", rb.Name, rb.Namespace)
				return nil
			}
			return err
		}

		rb, err = create(nil, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build RoleBinding : %v", err)
		}
		if equality.Semantic.DeepEqual(rb, existing) {
			continue
		}

		if err := r.client.Update(r.ctx, rb); err != nil {
			return fmt.Errorf("failed to update RoleBinding %s in namespace %s: %v", rb.Name, rb.Namespace, err)
		}
		glog.V(debugLevel).Infof("updated RoleBinding %s in namespace %s", rb.Name, rb.Namespace)
	}

	return nil
}

// GetUserClusterRoleCreators returns a list of GetUserClusterRoleCreators
func GetUserClusterRoleCreators() []resources.ClusterRoleCreator {
	creators := []resources.ClusterRoleCreator{
		machinecontroller.ClusterRole,
		vpnsidecar.DnatControllerClusterRole,
		metricsserver.ClusterRole,
		ipamcontroller.ClusterRole,
	}
	return creators
}

func (r *reconciler) ensureClusterEnsureRoles() error {
	creators := GetUserClusterRoleCreators()
	for _, create := range creators {
		cRole, err := create(nil, nil)
		if err != nil {
			return fmt.Errorf("failed to build ClusterRole: %v", err)
		}

		existing := &rbacv1.ClusterRole{}
		err = r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: cRole.Namespace, Name: cRole.Name}, existing)
		if err != nil {
			if errors.IsNotFound(err) {
				if err := r.client.Create(r.ctx, cRole); err != nil {
					return fmt.Errorf("failed to create ClusterRole %s in namespace %s: %v", cRole.Name, cRole.Namespace, err)
				}
				glog.V(debugLevel).Infof("created a new ClusterRole %s in namespace %s", cRole.Name, cRole.Namespace)
				return nil
			}
			return err
		}

		cRole, err = create(nil, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build ClusterRole : %v", err)
		}
		if equality.Semantic.DeepEqual(cRole, existing) {
			continue
		}

		if err := r.client.Update(r.ctx, cRole); err != nil {
			return fmt.Errorf("failed to update ClusterRole %s in namespace %s: %v", cRole.Name, cRole.Namespace, err)
		}
		glog.V(debugLevel).Infof("updated ClusterRole %s in namespace %s", cRole.Name, cRole.Namespace)
	}

	return nil
}

// GetUserClusterRoleBindingCreators returns a list of ClusterRoleBindingCreators which should be used to create user cluster role bindings.
func GetUserClusterRoleBindingCreators() []resources.ClusterRoleBindingCreator {
	creators := []resources.ClusterRoleBindingCreator{
		machinecontroller.ClusterRoleBinding,
		machinecontroller.NodeBootstrapperClusterRoleBinding,
		machinecontroller.NodeSignerClusterRoleBinding,
		vpnsidecar.DnatControllerClusterRoleBinding,
		metricsserver.ClusterRoleBindingResourceReader,
		metricsserver.ClusterRoleBindingAuthDelegator,
		scheduler.ClusterRoleBindingAuthDelegator,
		controllermanager.ClusterRoleBindingAuthDelegator,
		ipamcontroller.ClusterRoleBinding,
	}
	return creators
}

func (r *reconciler) ensureClusterEnsureRoleBindings() error {
	creators := GetUserClusterRoleBindingCreators()
	for _, create := range creators {
		crb, err := create(nil, nil)
		if err != nil {
			return fmt.Errorf("failed to build ClusterRoleBinding: %v", err)
		}

		existing := &rbacv1.ClusterRoleBinding{}
		err = r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: crb.Namespace, Name: crb.Name}, existing)
		if err != nil {
			if errors.IsNotFound(err) {
				if err := r.client.Create(r.ctx, crb); err != nil {
					return fmt.Errorf("failed to create ClusterRoleBinding %s in namespace %s: %v", crb.Name, crb.Namespace, err)
				}
				glog.V(debugLevel).Infof("created a new ClusterRoleBinding %s in namespace %s", crb.Name, crb.Namespace)
				return nil
			}
			return err
		}

		crb, err = create(nil, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build ClusterRoleBinding : %v", err)
		}
		if equality.Semantic.DeepEqual(crb, existing) {
			continue
		}

		if err := r.client.Update(r.ctx, crb); err != nil {
			return fmt.Errorf("failed to update ClusterRoleBinding %s in namespace %s: %v", crb.Name, crb.Namespace, err)
		}
		glog.V(debugLevel).Infof("updated ClusterRoleBinding %s in namespace %s", crb.Name, crb.Namespace)
	}

	return nil
}
