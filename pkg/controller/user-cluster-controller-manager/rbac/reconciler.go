/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rbacusercluster

import (
	"context"
	"fmt"

	controllerclient "sigs.k8s.io/controller-runtime/pkg/client"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

// reconciler creates and updates ClusterRoles and ClusterRoleBinding to achieve the desired state
type reconciler struct {
	ctx    context.Context
	client controllerclient.Client
}

// Reconcile creates and updates ClusterRoles and ClusterRoleBinding to achieve the desired state
func (r *reconciler) Reconcile(resourceName string) error {
	klog.V(4).Infof("Reconciling RBAC for %s", resourceName)

	err := r.ensureRBACClusterRole(resourceName)
	if err != nil {
		return err
	}
	err = r.ensureRBACClusterRoleBinding(resourceName)
	if err != nil {
		return err
	}

	return nil
}

func (r *reconciler) ensureRBACClusterRole(resourceName string) error {
	defaultClusterRole, err := GenerateRBACClusterRole(resourceName)
	if err != nil {
		return fmt.Errorf("failed to generate the RBAC Cluster Role: %v", err)
	}

	clusterRole := &rbacv1.ClusterRole{}
	err = r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: metav1.NamespaceAll, Name: resourceName}, clusterRole)
	if err != nil {
		// create Cluster Role if not exist
		if errors.IsNotFound(err) {
			if err := r.client.Create(r.ctx, defaultClusterRole); err != nil {
				return fmt.Errorf("failed to create the RBAC Cluster Role: %v", err)
			}
			klog.V(2).Infof("Created a new ClusterRole %s", defaultClusterRole.Name)
			return nil
		}
		return err
	}
	// compare Cluster Role with default. If don't match update for default
	if !ClusterRoleMatches(clusterRole, defaultClusterRole) {
		if err := r.client.Update(r.ctx, defaultClusterRole); err != nil {
			return fmt.Errorf("failed to update the RBAC Cluster Role: %v", err)
		}
		klog.V(2).Infof("Updated the ClusterCole %s", defaultClusterRole.Name)
	}

	return nil
}

func (r *reconciler) ensureRBACClusterRoleBinding(resourceName string) error {
	defaultClusterBinding, err := GenerateRBACClusterRoleBinding(resourceName)
	if err != nil {
		return fmt.Errorf("failed to generate the RBAC Cluster Role Binding: %v", err)
	}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{}
	if err := r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: metav1.NamespaceAll, Name: resourceName}, clusterRoleBinding); err != nil {
		if errors.IsNotFound(err) {
			if err := r.client.Create(r.ctx, defaultClusterBinding); err != nil {
				return fmt.Errorf("failed to create the RBAC ClusterRoleBinding: %v", err)
			}
			klog.V(2).Infof("Created a new ClusterRoleBinding %s", defaultClusterBinding.Name)
			return nil
		}
		return err
	}

	// compare cluster role bindings with default. If don't match update for default
	if !ClusterRoleBindingMatches(clusterRoleBinding, defaultClusterBinding) {
		if err := r.client.Update(r.ctx, defaultClusterBinding); err != nil {
			return fmt.Errorf("failed to update the RBAC ClusterRoleBinding: %v", err)
		}
		klog.V(2).Infof("Updated the ClusterColeBinding %s", defaultClusterBinding.Name)
	}
	return nil
}
