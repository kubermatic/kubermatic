package usercluster

import (
	"fmt"

	"github.com/golang/glog"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/ipamcontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/kubestatemetrics"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machinecontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/openvpn"
	"github.com/kubermatic/kubermatic/api/pkg/resources/vpnsidecar"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Get data needed to seed the user-cluster. This might even connect to the seed-cluster.
// For now it only accesses user-cluster.
func (ucc *Controller) userClusterEnsureClusterData() error {

	data, err := ucc.getUserClusterData()
	if err != nil {
		return err
	}

	// get some initial information from cluster-info
	name, err := data.GetClusterName()
	if err != nil {
		return fmt.Errorf("failed to get user-cluster name: %v", err)
	}

	if len(name) == 0 {
		return fmt.Errorf("empty user-cluster name")
	}

	return nil
}

// GetUserClusterRoleCreators returns a list of GetUserClusterRoleCreators
func GetUserClusterRoleCreators(data *resources.UserClusterData) []resources.UserClusterRoleCreator {
	creators := []resources.UserClusterRoleCreator{
		machinecontroller.ClusterRole,
		kubestatemetrics.ClusterRole,
		vpnsidecar.DnatControllerClusterRole,
	}

	if data.IpamEnabled() {
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

func (ucc *Controller) userClusterEnsureConfigMaps() error {
	creators := []resources.UserConfigMapCreator{
		openvpn.ClientConfigConfigMap,
	}

	data, err := ucc.getUserClusterData()
	if err != nil {
		return err
	}

	for _, create := range creators {
		var existing *corev1.ConfigMap
		cm, err := create(data, nil)
		if err != nil {
			return fmt.Errorf("failed to build ConfigMap: %v", err)
		}

		if existing, err = ucc.client.CoreV1().ConfigMaps(cm.Namespace).Get(cm.Name, metav1.GetOptions{}); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = ucc.client.CoreV1().ConfigMaps(cm.Namespace).Create(cm); err != nil {
				return fmt.Errorf("failed to create ConfigMap %s: %v", cm.Name, err)
			}
			glog.V(4).Infof("Created ConfigMap %s inside user-cluster %s", cm.Name, data.ClusterNameOrEmpty())
			continue
		}

		cm, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build ConfigMap: %v", err)
		}

		if equality.Semantic.DeepEqual(cm, existing) {
			continue
		}

		if _, err = ucc.client.CoreV1().ConfigMaps(cm.Namespace).Update(cm); err != nil {
			return fmt.Errorf("failed to update ConfigMap %s: %v", cm.Name, err)
		}
		glog.V(4).Infof("Updated ConfigMap %s inside user-cluster %s", cm.Name, data.ClusterNameOrEmpty())
	}

	return nil
}

func (ucc *Controller) getUserClusterData() (*resources.UserClusterData, error) {
	return resources.NewUserClusterData(ucc.configMapLister), nil
}
