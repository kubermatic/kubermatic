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

package common

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/install/helm"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NginxIngressControllerChartName   = "nginx-ingress-controller"
	NginxIngressControllerReleaseName = NginxIngressControllerChartName
	NginxIngressControllerNamespace   = NginxIngressControllerChartName
)

// NginxIngressNamespaceExists reports whether the legacy nginx-ingress-controller
// namespace is still present on the cluster.
func NginxIngressNamespaceExists(ctx context.Context, kubeClient ctrlruntimeclient.Client) (bool, error) {
	ns := &corev1.Namespace{}
	if err := kubeClient.Get(ctx, types.NamespacedName{Name: NginxIngressControllerNamespace}, ns); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to probe for %s namespace: %w", NginxIngressControllerNamespace, err)
	}
	return true, nil
}

// UninstallNginxIngressController removes the legacy nginx-ingress-controller Helm release
// and deletes its namespace. Both steps are best-effort and skipped silently when the
// release or namespace no longer exists.
func UninstallNginxIngressController(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client) error {
	exists, err := NginxIngressNamespaceExists(ctx, kubeClient)
	if err != nil {
		return err
	}
	if !exists {
		logger.Infof("⭕ %s namespace not present, nothing to clean up.", NginxIngressControllerNamespace)
		return nil
	}

	logger.Infof("🧹 Uninstalling legacy %s Helm release…", NginxIngressControllerChartName)
	if err := helmClient.UninstallRelease(NginxIngressControllerNamespace, NginxIngressControllerReleaseName); err != nil {
		return fmt.Errorf("failed to uninstall %s Helm release: %w", NginxIngressControllerReleaseName, err)
	}

	ns := &corev1.Namespace{}
	if err := kubeClient.Get(ctx, types.NamespacedName{Name: NginxIngressControllerNamespace}, ns); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get %s namespace: %w", NginxIngressControllerNamespace, err)
	}
	if err := kubeClient.Delete(ctx, ns); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s namespace: %w", NginxIngressControllerNamespace, err)
	}

	logger.Info("✅ nginx-ingress-controller removed.")
	return nil
}
