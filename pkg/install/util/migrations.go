/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package util

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ShutdownDeployment takes the name and namespace of a deployment and will scale that
// Deployment to 0 replicas.
func ShutdownDeployment(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, namespace string, name string) error {
	depLogger := logger.WithField("deployment", name).WithField("namespace", namespace)

	deployment := appsv1.Deployment{}
	key := types.NamespacedName{Name: name, Namespace: namespace}

	if err := client.Get(ctx, key, &deployment); err != nil {
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

// WaitForAllPodsToBeGone takes the name of a Deployment and will wait until all Pods for that
// Deployment have been shut down.
func WaitForAllPodsToBeGone(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, namespace string, name string, timeout time.Duration) error {
	depLogger := logger.WithField("deployment", name).WithField("namespace", namespace)
	depLogger.Debug("Waiting for Pods to be gone…")

	// waiting for shutdown, as even a Terminating pod can still run and do dangerous things;
	// sadly Kubernetes does not provide any status information on the Deployment when it's
	// scaled down to 0 replicas, so we must check for existing pods
	podNamePrefix := name + "-"

	err := wait.Poll(500*time.Millisecond, timeout, func() (done bool, err error) {
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

func RemoveValidatingWebhook(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, webhookName string) error {
	webhook := admissionregistrationv1.ValidatingWebhookConfiguration{}
	webhook.Name = webhookName

	return removeWebhook(ctx, logger.WithField("webhook", webhookName), client, &webhook)
}

func RemoveMutatingWebhook(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, webhookName string) error {
	webhook := admissionregistrationv1.MutatingWebhookConfiguration{}
	webhook.Name = webhookName

	return removeWebhook(ctx, logger.WithField("webhook", webhookName), client, &webhook)
}

func removeWebhook(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, obj ctrlruntimeclient.Object) error {
	if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(obj), obj); err != nil {
		// not all webhooks need to exist in all clusters / maybe we already cleaned up
		// because the user ran the "shutdown" command twice
		if apierrors.IsNotFound(err) {
			logger.Debug("Webhook not found.")
			return nil
		}

		return fmt.Errorf("failed to get webhook %s: %w", obj.GetName(), err)
	}

	logger.Debug("Removing…")

	if err := client.Delete(ctx, obj); err != nil {
		return fmt.Errorf("failed to remove webhook %s: %w", obj.GetName(), err)
	}

	return nil
}

// RemoveOldApplicationDefinition removes old applicationDefinitions from cluster.
// See pkg/install/stack/kubermatic-master/stack.go::InstallKubermaticCRDs() for more information.
// TODO REMOVE AFTER release v2.21.
func RemoveOldApplicationDefinition(ctx context.Context, kubeClient ctrlruntimeclient.Client) error {
	crd := &apiextensionsv1.CustomResourceDefinition{}
	err := kubeClient.Get(ctx, types.NamespacedName{Name: "applicationdefinitions.apps.kubermatic.k8c.io"}, crd)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to get CRD applicationdefinitions.apps.kubermatic.k8c.io: %w", err)
	}

	// No action is required
	if err != nil && apierrors.IsNotFound(err) {
		return nil
	}

	unstructuredList := unstructured.UnstructuredList{}
	unstructuredList.SetKind("ApplicationDefinition")
	unstructuredList.SetAPIVersion("apps.kubermatic.k8c.io/v1")
	if err := kubeClient.List(ctx, &unstructuredList); err != nil {
		return fmt.Errorf("failed to get applicationDefition: %w", err)
	}

	for _, unstruturedAppDef := range unstructuredList.Items {
		appDef := &appskubermaticv1.ApplicationDefinition{}

		// Remove applicationDefinitio if it's not valid ortherwise keep it.
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstruturedAppDef.UnstructuredContent(), &appDef); err != nil {
			if err := kubeClient.Delete(ctx, &unstruturedAppDef); err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to delete applicationDefinition %s: %w", unstruturedAppDef.GetName(), err)
			}
		}

		// this filed is mandatory and immutable in new version
		if appDef.Spec.Method == "" {
			if err := kubeClient.Delete(ctx, &unstruturedAppDef); err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to delete applicationDefinition %s: %w", unstruturedAppDef.GetName(), err)
			}
		}
	}
	return nil
}
