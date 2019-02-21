package rolebinding

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"
	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac-user-cluster"

	controllerclient "sigs.k8s.io/controller-runtime/pkg/client"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	debugLevel = 4
)

// reconciler creates and updates ClusterRoleBinding to achieve the desired state
type reconciler struct {
	ctx    context.Context
	client controllerclient.Client
}

// reconcile creates and updates ClusterRoleBinding to achieve the desired state
func (r *reconciler) Reconcile() error {

	for _, groupName := range rbac.AllGroupsPrefixes {

		err := r.ensureRBACClusterRoleBinding(groupName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *reconciler) ensureRBACClusterRoleBinding(groupName string) error {
	defaultClusterBinding := rbacusercluster.GenerateRBACClusterRoleBinding(groupName)

	// check if ClusterRole already exists to make binding
	clusterRole := &rbacv1.ClusterRole{}
	// if the corresponding ClusterRole does not exit then the reconciler return an error and the request will be retried.
	if err := r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: metav1.NamespaceAll, Name: defaultClusterBinding.Name}, clusterRole); err != nil {
		glog.Error("unable to create ClusterRoleBinding because the corresponding ClusterRole doesn't exist")
		return err
	}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{}
	if err := r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: metav1.NamespaceAll, Name: defaultClusterBinding.Name}, clusterRoleBinding); err != nil {
		if errors.IsNotFound(err) {
			glog.V(debugLevel).Infof("creating a new ClusterRoleBinding %s", defaultClusterBinding.Name)
			if err := r.client.Create(r.ctx, defaultClusterBinding); err != nil {
				return fmt.Errorf("failed to create the RBAC ClusterRoleBinding: %v", err)
			}
			return nil
		}
		return err
	}

	// compare cluster role bindings with default. If don't match update for default
	if !rbacusercluster.ClusterRoleBindingMatches(clusterRoleBinding, defaultClusterBinding) {
		glog.V(debugLevel).Infof("updating the ClusterRoleBidning %s because it doesn't match the expected one", defaultClusterBinding.Name)
		if err := r.client.Update(r.ctx, defaultClusterBinding); err != nil {
			return fmt.Errorf("failed to update the RBAC ClusterRoleBinding: %v", err)
		}
	}
	return nil
}
