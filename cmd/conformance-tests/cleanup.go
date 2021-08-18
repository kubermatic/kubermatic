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

package main

import (
	"context"

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/resources"
	kubernetesdashboard "k8c.io/kubermatic/v2/pkg/resources/kubernetes-dashboard"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	protectedNamespaces = sets.NewString(
		metav1.NamespaceDefault,
		metav1.NamespaceSystem,
		metav1.NamespacePublic,
		corev1.NamespaceNodeLease,
		kubernetesdashboard.Namespace,
		resources.CloudInitSettingsNamespace,
	)
)

func (r *testRunner) cleanupBeforeGinkgo(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client) error {
	log.Info("Removing webhooks...")

	if err := wait.Poll(r.userClusterPollInterval, r.customTestTimeout, func() (done bool, err error) {
		webhookList := &admissionregistrationv1.ValidatingWebhookConfigurationList{}
		if err := client.List(ctx, webhookList); err != nil {
			log.Errorw("Failed to list webhooks", zap.Error(err))
			return false, nil
		}

		if len(webhookList.Items) == 0 {
			return true, nil
		}

		for _, webhook := range webhookList.Items {
			if webhook.DeletionTimestamp == nil {
				wlog := log.With("webhook", webhook.Name)

				if err := client.Delete(ctx, &webhook); err != nil {
					wlog.Errorw("Failed to delete webhook", zap.Error(err))
				} else {
					wlog.Debug("Deleted webhook.")
				}
			}
		}

		return false, nil
	}); err != nil {
		return err
	}

	log.Info("Removing non-default namespaces...")

	return wait.Poll(r.userClusterPollInterval, r.customTestTimeout, func() (done bool, err error) {
		namespaceList := &corev1.NamespaceList{}
		if err := client.List(ctx, namespaceList); err != nil {
			log.Errorw("Failed to list namespaces", zap.Error(err))
			return false, nil
		}

		// This check assumes no one deleted one of the protected namespaces
		if len(namespaceList.Items) <= protectedNamespaces.Len() {
			return true, nil
		}

		for _, namespace := range namespaceList.Items {
			if protectedNamespaces.Has(namespace.Name) {
				continue
			}

			// If it's not gone & the DeletionTimestamp is nil, delete it
			if namespace.DeletionTimestamp == nil {
				nslog := log.With("namespace", namespace.Name)

				if err := client.Delete(ctx, &namespace); err != nil {
					nslog.Errorw("Failed to delete namespace", zap.Error(err))
				} else {
					nslog.Debug("Deleted namespace.")
				}
			}
		}

		return false, nil
	})
}
