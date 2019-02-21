package role

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

// reconciler creates and updates ClusterRoles to achieve the desired state
type reconciler struct {
	ctx    context.Context
	client controllerclient.Client
}

// Reconcile creates and updates ClusterRoles to achieve the desired state
func (r *reconciler) Reconcile() error {

	for _, groupName := range rbac.AllGroupsPrefixes {

		err := r.ensureRBACClusterRole(groupName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *reconciler) ensureRBACClusterRole(groupName string) error {
	defaultClusterRole, err := rbacusercluster.GenerateRBACClusterRole(groupName)
	if err != nil {
		return fmt.Errorf("failed to generate the RBAC Cluster Role: %v", err)
	}

	clusterRole := &rbacv1.ClusterRole{}
	err = r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: metav1.NamespaceAll, Name: defaultClusterRole.Name}, clusterRole)
	if err != nil {
		// create Cluster Role if not exist
		if errors.IsNotFound(err) {
			glog.V(debugLevel).Infof("creating a new Cluster Role %s", defaultClusterRole.Name)
			if err := r.client.Create(r.ctx, defaultClusterRole); err != nil {
				return fmt.Errorf("failed to create the RBAC Cluster Role: %v", err)
			}
			return nil
		}
		return err
	}
	// compare Cluster Role with default. If don't match update for default
	if !rbacusercluster.ClusterRoleMatches(clusterRole, defaultClusterRole) {
		glog.V(debugLevel).Infof("updating the Cluster Role %s because it doesn't match the expected one", defaultClusterRole.Name)
		if err := r.client.Update(r.ctx, defaultClusterRole); err != nil {
			return fmt.Errorf("failed to update the RBAC Cluster Role: %v", err)
		}
	}

	return nil
}
