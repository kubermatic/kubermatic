package usercluster

import (
	"fmt"

	"github.com/golang/glog"

	// kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/controller/cluster"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/ipamcontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/kubestatemetrics"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machinecontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/vpnsidecar"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetUserClusterRoleCreators returns a list of GetUserClusterRoleCreators
func GetUserClusterRoleCreators(data *resources.UserClusterData) []resources.UserClusterRoleCreator {
	creators := []resources.UserClusterRoleCreator{
		machinecontroller.ClusterRole,
		kubestatemetrics.ClusterRole,
		vpnsidecar.DnatControllerClusterRole,
	}

	if len(seed.SpecMachineNetworks) > 0 {
		creators = append(creators, ipamcontroller.ClusterRole)
	}

	return creators
}

func (ucc *Controller) userClusterEnsureClusterRoles() error {

	data, err := ucc.getUserClusterData()
	if err != nil {
		return err
	}

	creators := GetUserClusterRoleCreators(data)

	for _, create := range creators {
		var existing *rbacv1.ClusterRole
		cRole, err := create(data, nil)
		if err != nil {
			return fmt.Errorf("failed to build ClusterRole: %v", err)
		}

		if existing, err = ucc.client.RbacV1().ClusterRoles().Get(cRole.Name, metav1.GetOptions{}); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = ucc.client.RbacV1().ClusterRoles().Create(cRole); err != nil {
				return fmt.Errorf("failed to create ClusterRole %s: %v", cRole.Name, err)
			}
			glog.V(4).Infof("Created ClusterRole %s inside user-cluster %s", cRole.Name, data.ClusterNameOrEmpty())
			continue
		}

		cRole, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build ClusterRole: %v", err)
		}

		if equality.Semantic.DeepEqual(cRole, existing) {
			continue
		}

		if _, err = ucc.client.RbacV1().ClusterRoles().Update(cRole); err != nil {
			return fmt.Errorf("failed to update ClusterRole %s: %v", cRole.Name, err)
		}
		glog.V(4).Infof("Updated ClusterRole %s inside user-cluster %s", cRole.Name, data.ClusterNameOrEmpty())
	}

	return nil
}
func (ucc *Controller) getUserClusterData() (*resources.UserClusterData, error) {
	return resources.NewUserClusterData(ucc.configMapLister), nil
}
