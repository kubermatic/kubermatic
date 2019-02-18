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

// reconciler creates and deletes Kubernetes resources to achieve the desired state of an RBAC Definition
type reconciler struct {
	ctx    context.Context
	client controllerclient.Client
}

// Reconcile creates, updates, or deletes Kubernetes resources to match
//   the desired state defined in an RBAC Definition
func (r *reconciler) Reconcile() error {

	for _, groupName := range rbac.AllGroupsPrefixes {

		err := r.ensureRBACRoleBinding(groupName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *reconciler) ensureRBACRoleBinding(groupName string) error {
	defaultClusterBinding := rbacusercluster.GenerateRBACClusterRoleBinding(groupName)

	// check if cluster role already exists to make binding
	clusterRole := &rbacv1.ClusterRole{}
	if err := r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: metav1.NamespaceAll, Name: defaultClusterBinding.Name}, clusterRole); err != nil {
		glog.Error("getting  cluster role ", defaultClusterBinding.Name, " failed with error: ", err)
		return err
	}

	clusterRoleBindings := &rbacv1.ClusterRoleBinding{}
	if err := r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: metav1.NamespaceAll, Name: defaultClusterBinding.Name}, clusterRoleBindings); err != nil {
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
	if !rbacusercluster.ClusterRoleBindingsMatches(clusterRoleBindings, defaultClusterBinding) {
		glog.V(debugLevel).Infof("updating the ClusterRoleBidning %s because it doesn't match the expected one", defaultClusterBinding.Name)
		if err := r.client.Update(r.ctx, defaultClusterBinding); err != nil {
			return fmt.Errorf("failed to update the RBAC ClusterRoleBinding: %v", err)
		}
	}
	return nil
}
