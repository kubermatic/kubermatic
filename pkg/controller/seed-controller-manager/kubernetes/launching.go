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

package kubernetes

import (
	"context"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/etcd"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// clusterIsReachable checks if the cluster is reachable via its external name.
func (r *Reconciler) clusterIsReachable(ctx context.Context, c *kubermaticv1.Cluster) (bool, error) {
	client, err := r.userClusterConnProvider.GetClient(ctx, c)
	if err != nil {
		return false, err
	}

	if err := client.List(ctx, &corev1.NamespaceList{}); err != nil {
		r.log.Debugw("Cluster not yet reachable", "cluster", c.Name, zap.Error(err))
		return false, nil
	}

	return true, nil
}

func (r *Reconciler) etcdUseStrictTLS(ctx context.Context, c *kubermaticv1.Cluster) (bool, error) {
	statefulSet := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Namespace: c.Status.NamespaceName, Name: resources.EtcdStatefulSetName}, statefulSet)

	if err != nil {
		// if the StatefulSet for etcd doesn't exist yet, a new one can be deployed with strict TLS peers
		if apierrors.IsNotFound(err) {
			return true, nil
		} else {
			return false, err
		}
	}

	pods := &corev1.PodList{}
	labelSet := etcd.GetBasePodLabels(c)

	err = r.List(ctx, pods, &ctrlruntimeclient.ListOptions{
		Namespace:     c.Status.NamespaceName,
		LabelSelector: labels.SelectorFromSet(labelSet),
	})

	if err != nil {
		return false, err
	}

	for _, pod := range pods.Items {
		if _, ok := pod.Annotations[resources.EtcdTLSEnabledAnnotation]; !ok {
			return false, nil
		}
	}

	return true, nil
}
