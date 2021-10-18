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

	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	kubermaticmaster "k8c.io/kubermatic/v2/pkg/controller/operator/master/resources/kubermatic"
	kubermaticseed "k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/kubermatic"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/crd/util"
	"k8c.io/kubermatic/v2/pkg/resources"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func PerformPreflightChecks(ctx context.Context, logger logrus.FieldLogger, opt *Options) error {
	success := true

	if err := validateSeedClients(ctx, logger, opt); err != nil {
		logger.Errorf("Seed client validation failed: %v", err)
		success = false
	}

	if err := validateKubermaticNotRunning(ctx, logger, opt); err != nil {
		logger.Errorf("KKP health check failed: %v", err)
		success = false
	}

	if err := validateNoStuckResources(ctx, logger, opt); err != nil {
		logger.Errorf("Resource health check failed: %v", err)
		success = false
	}

	if err := validateCRDsExist(ctx, logger, opt); err != nil {
		logger.Errorf("CustomResourceDefinition check failed: %v", err)
		success = false
	}

	if !success {
		return errors.New("please correct the errors noted above and try again")
	}

	return nil
}

// validateSeedClients checks if the clients actually work. To ensure
// this, we simply check if we can retrieve the kubermatic namespace
// which must exist on all master and seed clusters.
func validateSeedClients(ctx context.Context, logger logrus.FieldLogger, opt *Options) error {
	logger.Info("Validating seed clients…")

	key := types.NamespacedName{
		Name: opt.KubermaticNamespace,
	}

	success := true

	for seedName, seedClient := range opt.SeedClients {
		seedLogger := logger.WithField("seed", seedName)
		seedLogger.Debug("Validating…")

		ns := corev1.Namespace{}
		if err := seedClient.Get(ctx, key, &ns); err != nil {
			success = false

			if apierrors.IsNotFound(err) {
				seedLogger.Warnf("No %s namespace exists on this cluster.", key.Name)
			} else {
				seedLogger.Warnf("Failed to check that %q namespace exists: %v", key.Name, err)
			}
		}
	}

	if !success {
		return errors.New("one or more of the seed clients is defunct, please check that all Seed resources have a working kubeconfig attached")
	}

	return nil
}

func validateKubermaticNotRunning(ctx context.Context, logger logrus.FieldLogger, opt *Options) error {
	logger.Info("Validating that KKP is not running…")

	success := true

	// check master cluster
	if !validateKubermaticNotRunningInCluster(ctx, logger.WithField("master", true), opt.MasterClient, opt, false) {
		success = false
	}

	// check seeds
	for seedName, seedClient := range opt.SeedClients {
		if !validateKubermaticNotRunningInCluster(ctx, logger.WithField("seed", seedName), seedClient, opt, true) {
			success = false
		}
	}

	if !success {
		return errors.New("please scale down all KKP deployments to 0 and remove KKP webhooks")
	}

	return nil
}

func validateKubermaticNotRunningInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, opt *Options, isSeed bool) bool {
	logger.Info("Checking controllers…")

	success := true

	deployments := []string{
		"kubermatic-operator", // as named in our Helm chart
		common.MasterControllerManagerDeploymentName,
		kubermaticmaster.APIDeploymentName,
	}

	if isSeed {
		deployments = []string{common.SeedControllerManagerDeploymentName}
	}

	for _, name := range deployments {
		if !validateDeploymentHasNoReplicas(ctx, logger, client, opt.KubermaticNamespace, name) {
			success = false
		}
	}

	logger.Info("Checking webhooks…")

	webhooks := []string{
		kubermaticseed.ClusterAdmissionWebhookName,
		common.SeedAdmissionWebhookName(opt.KubermaticConfiguration),
	}

	for _, name := range webhooks {
		if !validateWebhookDoesNotExist(ctx, logger, client, name) {
			success = false
		}
	}

	// It would be harmless to check for userclusters on the master, as it
	// would simply find no namespaces, but on shared master/seed clusters,
	// this would lead to problems with userclusters reported twice.
	if isSeed {
		logger.Info("Checking userclusters…")

		clusterNamespaces, err := getUserclusterNamespaces(ctx, client)
		if err != nil {
			logger.Warnf("Failed to get namespaces: %v", err)
			success = false
		} else {
			for _, namespace := range clusterNamespaces {
				if !validateDeploymentHasNoReplicas(ctx, logger, client, namespace, resources.UserClusterControllerDeploymentName) {
					success = false
				}
			}
		}
	}

	return success
}

func validateDeploymentHasNoReplicas(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, namespace string, name string) bool {
	deployment := appsv1.Deployment{}
	key := types.NamespacedName{Name: name, Namespace: namespace}

	logger = logger.WithField("deployment", key)
	logger.Debug("Validating…")

	if err := client.Get(ctx, key, &deployment); err != nil {
		if apierrors.IsNotFound(err) {
			return true
		}

		logger.Warnf("Failed to retrieve Deployment: %v", err)
		return false
	}

	if replicas := deployment.Status.Replicas; replicas > 0 {
		if replicas == 1 {
			logger.Warnf("Deployment still has %d replica.", replicas)
		} else {
			logger.Warnf("Deployment still has %d replicas.", replicas)
		}
		return false
	}

	return true
}

func validateWebhookDoesNotExist(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, name string) bool {
	webhook := admissionregistrationv1.ValidatingWebhookConfiguration{}
	key := types.NamespacedName{Name: name}

	logger = logger.WithField("webhhook", name)
	logger.Debug("Validating…")

	if err := client.Get(ctx, key, &webhook); err != nil {
		if apierrors.IsNotFound(err) {
			return true
		}

		logger.Warnf("Failed to retrieve ValidatingWebhook: %v", err)
		return false
	}

	return false
}

func validateNoStuckResources(ctx context.Context, logger logrus.FieldLogger, opt *Options) error {
	logger.Info("Validating all KKP resources are healthy…")

	success := true

	// check master cluster
	masterResources := []string{
		"ClusterTemplate",
		"User",
		"UserProjectBinding",
		"UserSSHKey",
	}

	if !validateNoStuckResourcesInCluster(ctx, logger.WithField("master", true), opt.MasterClient, masterResources) {
		success = false
	}

	// check seed clusters
	seedResources := []string{
		"Cluster",
	}

	for seedName, seedClient := range opt.SeedClients {
		seedLogger := logger.WithField("seed", seedName)

		if !validateNoStuckResourcesInCluster(ctx, seedLogger, seedClient, seedResources) {
			success = false
		}

		// check cluster namespaces as well
		clusterNamespaces, err := getUserclusterNamespaces(ctx, seedClient)
		if err != nil {
			seedLogger.Warnf("Failed to get namespaces: %v", err)
			success = false
		} else {
			for _, namespace := range clusterNamespaces {
				nsLogger := seedLogger.WithField("namespace", namespace)
				nsLogger.Debug("Validating…")

				ns := corev1.Namespace{}
				key := types.NamespacedName{Name: namespace}

				if err := seedClient.Get(ctx, key, &ns); err != nil {
					nsLogger.Warnf("Failed to get namespaces: %v", err)
					success = false
				} else if ns.DeletionTimestamp != nil {
					nsLogger.Warn("Namespace is in deletion.")
					success = false
				}
			}
		}
	}

	if !success {
		return errors.New("please ensure that no KKP resource is stuck before continuing")
	}

	return nil
}

func validateNoStuckResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, kinds []string) bool {
	success := true

	for _, kind := range kinds {
		objectList := &metav1unstructured.UnstructuredList{}
		objectList.SetAPIVersion(kubermaticv1.SchemeGroupVersion.String())
		objectList.SetKind(kind)

		if err := client.List(ctx, objectList); err != nil {
			logger.Warnf("Failed to list %s objects: %v", kind, err)
			success = false
			continue
		}

		for _, object := range objectList.Items {
			objectLogger := logger.WithField(strings.ToLower(object.GetKind()), object.GetName())
			objectLogger.Debug("Validating…")

			if object.GetDeletionTimestamp() != nil {
				objectLogger.Warnf("%s is in deletion.", kind)
				success = false
			}
		}
	}

	return success
}

// validateCRDsExist checks that the installer has access to the
// *new* CRDs. This is mostly a sanity check to prevent users from
// running the installer in weird ways or accidentally mix it with
// the old CRDs.
// The installer will later during the migration install/update the
// CRDs itself and at that point we want to be sure everything's OK.
func validateCRDsExist(ctx context.Context, logger logrus.FieldLogger, opt *Options) error {
	logger.Info("Validating all new KKP CRDs exist…")

	crdDirectory := "charts/kubermatic-operator/crd"

	crds, err := util.LoadFromDirectory(crdDirectory)
	if err != nil {
		return fmt.Errorf("failed to load CRDs from %s: %w", crdDirectory, err)
	}

	checklist := sets.NewString(allKubermaticKinds...)

	for _, crd := range crds {
		// not actually a CRD
		if crd.GetObjectKind().GroupVersionKind() != apiextensionsv1.SchemeGroupVersion.WithKind("CustomResourceDefinition") {
			continue
		}

		crdUnstructured, ok := crd.(*unstructured.Unstructured)
		if ok {
			var crdObject apiextensionsv1.CustomResourceDefinition

			// after the GVK check, this should only happen if someone manually breaks the YAML
			if err = runtime.DefaultUnstructuredConverter.FromUnstructured(crdUnstructured.Object, &crdObject); err != nil {
				continue
			}

			// This is important, we need to ensure that we actually found
			// CRDs for the *new* API group.
			if crdObject.Spec.Group != newAPIGroup {
				continue
			}

			checklist.Delete(crdObject.Spec.Names.Kind)
		}
	}

	if checklist.Len() > 0 {
		return fmt.Errorf("could not find files containing the CRDs for %v", checklist.List())
	}

	return nil
}
