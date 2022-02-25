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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	kubermaticmaster "k8c.io/kubermatic/v2/pkg/controller/operator/master/resources/kubermatic"
	kubermaticseed "k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/kubermatic"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/resources"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
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
	// shutdown operator first, it would recreate everything else
	deploymentName := "kubermatic-operator" // as named in our Helm chart

	if err := shutdownDeployment(ctx, logger, client, kubermaticNamespace, deploymentName); err != nil {
		return err
	}

	if err := waitForAllPodsToBeGone(ctx, logger, client, kubermaticNamespace, deploymentName); err != nil {
		return err
	}

	// now shut down KKP controllers
	deployments := []string{
		common.MasterControllerManagerDeploymentName,
		kubermaticmaster.APIDeploymentName,
		common.SeedControllerManagerDeploymentName,
	}

	for _, deploymentName := range deployments {
		if err := shutdownDeployment(ctx, logger, client, kubermaticNamespace, deploymentName); err != nil {
			return err
		}
	}

	for _, deploymentName := range deployments {
		if err := waitForAllPodsToBeGone(ctx, logger, client, kubermaticNamespace, deploymentName); err != nil {
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

		for _, namespace := range clusterNamespaces {
			if err := waitForAllPodsToBeGone(ctx, logger, client, namespace, resources.UserClusterControllerDeploymentName); err != nil {
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

func waitForAllPodsToBeGone(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, namespace string, name string) error {
	depLogger := logger.WithField("deployment", name).WithField("namespace", namespace)
	depLogger.Debug("Waiting for Pods to be gone…")

	// waiting for shutdown, as even a Terminating pod can still run and do dangerous things;
	// sadly Kubernetes does not provide any status information on the Deployment when it's
	// scaled down to 0 replicas, so we must check for existing pods
	podNamePrefix := name + "-"

	err := wait.Poll(500*time.Millisecond, 1*time.Minute, func() (done bool, err error) {
		pods := &corev1.PodList{}
		opt := ctrlruntimeclient.ListOptions{
			Namespace: namespace,
		}

		if err := client.List(ctx, pods, &opt); err != nil {
			return false, fmt.Errorf("failed to list pods in %s: %w", namespace, err)
		}

		// Kubernetes does not provide real status information for pods that are terminating,
		// so all we have to go on is pod existence, which in itself can be problematic on
		// some providers like GKE which like to keep Terminated pods around forever.
		for _, pod := range pods.Items {
			// we found a pod
			if strings.HasPrefix(pod.Name, podNamePrefix) {
				return false, nil
			}
		}

		// no more pods left
		return true, nil
	})

	if errors.Is(err, wait.ErrWaitTimeout) {
		return errors.New("there are still Pods running, please wait and let them shut down")
	}

	return err
}

func shutdownWebhooksInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, config *operatorv1alpha1.KubermaticConfiguration) error {
	if err := shutdownValidatingWebhooksInCluster(ctx, logger, client, config); err != nil {
		return err
	}

	if err := shutdownMutatingWebhooksInCluster(ctx, logger, client, config); err != nil {
		return err
	}

	return nil
}

func shutdownValidatingWebhooksInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, config *operatorv1alpha1.KubermaticConfiguration) error {
	webhooks := []string{
		kubermaticseed.ClusterAdmissionWebhookName,
		// this cheats a bit and assumes that the function only needs the object meta
		common.SeedAdmissionWebhookName(&kubermaticv1.KubermaticConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.Name,
				Namespace: config.Namespace,
			},
		}),
	}

	if config.Spec.FeatureGates.Has(features.OperatingSystemManager) {
		webhooks = append(webhooks, kubermaticseed.OSCAdmissionWebhookName)
		webhooks = append(webhooks, kubermaticseed.OSPAdmissionWebhookName)
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

func shutdownMutatingWebhooksInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, config *operatorv1alpha1.KubermaticConfiguration) error {
	webhooks := []string{
		kubermaticseed.ClusterAdmissionWebhookName,
	}

	for _, webhookName := range webhooks {
		hookLogger := logger.WithField("webhook", webhookName)

		webhook := admissionregistrationv1.MutatingWebhookConfiguration{}
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
