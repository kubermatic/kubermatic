/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package crdmigration

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	kubermaticmaster "k8c.io/kubermatic/v2/pkg/controller/operator/master/resources/kubermatic"
	kubermaticseed "k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/kubermatic"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/resources"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func ShutdownControllers(ctx context.Context, logger logrus.FieldLogger, opt *Options) error {
	// master cluster
	if err := shutdownCluster(ctx, logger.WithField("master", true), opt.MasterClient, opt, false); err != nil {
		return fmt.Errorf("shutting down the master cluster failed: %w", err)
	}

	// seed clusters
	for seedName, seedClient := range opt.SeedClients {
		if err := shutdownCluster(ctx, logger.WithField("seed", seedName), seedClient, opt, true); err != nil {
			return fmt.Errorf("shutting down the seed cluster failed: %w", err)
		}
	}

	return nil
}

func shutdownCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, opt *Options, isSeed bool) error {
	logger.Info("Shutting down in cluster…")

	if err := shutdownDeploymentsInCluster(ctx, logger, client, opt.KubermaticNamespace, isSeed); err != nil {
		return err
	}

	if err := shutdownWebhooksInCluster(ctx, logger, client, opt.KubermaticConfiguration); err != nil {
		return err
	}

	return nil
}

func shutdownDeploymentsInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, kubermaticNamespace string, isSeed bool) error {
	deployments := []string{
		"kubermatic-operator", // as named in our Helm chart
		common.MasterControllerManagerDeploymentName,
		kubermaticmaster.APIDeploymentName,
		common.SeedControllerManagerDeploymentName,
	}

	for _, deploymentName := range deployments {
		if err := shutdownDeployment(ctx, logger, client, kubermaticNamespace, deploymentName); err != nil {
			return err
		}
	}

	// It would be harmless to check for userclusters on the master, as it
	// would simply find no namespaces, but on shared master/seed clusters,
	// this would lead to problems with userclusters reported twice.
	if isSeed {
		clusterNamespaces, err := getUserclusterNamespaces(ctx, client)
		if err != nil {
			return err
		}

		for _, namespace := range clusterNamespaces {
			if err := shutdownDeployment(ctx, logger, client, namespace, resources.UserClusterControllerDeploymentName); err != nil {
				return err
			}
		}
	}

	return nil
}

func shutdownDeployment(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, namespace string, name string) error {
	depLogger := logger.WithField("deployment", name).WithField("namespace", namespace)

	deployment := appsv1.Deployment{}
	key := types.NamespacedName{Name: name, Namespace: namespace}

	if err := client.Get(ctx, key, &deployment); err != nil {
		// not all deployments need to exist in all clusters
		if apierrors.IsNotFound(err) {
			depLogger.Debug("Deployment not found.")
			return nil
		}

		return fmt.Errorf("failed to get Deployment %s: %w", name, err)
	}

	if deployment.Spec.Replicas != nil && *deployment.Spec.Replicas > 0 {
		depLogger.Debug("Scaling down…")

		oldDeployment := deployment.DeepCopy()
		deployment.Spec.Replicas = pointer.Int32(0)

		if err := client.Patch(ctx, &deployment, ctrlruntimeclient.MergeFrom(oldDeployment)); err != nil {
			return fmt.Errorf("failed to scale down Deployment %s: %w", name, err)
		}
	}

	return nil
}

func shutdownWebhooksInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, config *operatorv1alpha1.KubermaticConfiguration) error {
	webhooks := []string{
		kubermaticseed.ClusterAdmissionWebhookName,
		common.SeedAdmissionWebhookName(config),
	}

	for _, webhookName := range webhooks {
		hookLogger := logger.WithField("webhook", webhookName)

		webhook := admissionregistrationv1.ValidatingWebhookConfiguration{}
		key := types.NamespacedName{Name: webhookName}

		if err := client.Get(ctx, key, &webhook); err != nil {
			// not all webhooks need to exist in all clusters / maybe we already cleaned up
			// because the user ran the "shutdown" command twice
			if apierrors.IsNotFound(err) {
				hookLogger.Debug("Webhook not found.")
				continue
			}

			return fmt.Errorf("failed to get Webhook %s: %w", webhookName, err)
		}

		hookLogger.Debug("Removing…")

		if err := client.Delete(ctx, &webhook); err != nil {
			return fmt.Errorf("failed to remove Webhook %s: %w", webhookName, err)
		}
	}

	return nil
}
