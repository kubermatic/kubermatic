package crdmigration

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/kubermatic"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// phase 1: preflight checks

// task 1.1: collect and check Seeds and verify that we can connect to all seed clusters
// task 1.2: check that all KKP controllers in all clusters are down, including usercluster-cm
// task 1.3: check that all KKP webhooks have been deleted
// task 1.4: check that no clusters (Cluster objects and cluster namespaces)
//           are stuck in deletion
// task 1.5: make sure the user explicitly confirms that they want to migrate now, e.g.
//           using --i-am-ready-now or --lets-get-dangerous

func PerformPreflightChecks(ctx context.Context, logger logrus.FieldLogger, opt *Options) error {
	if err := validateSeedClients(ctx, logger, opt); err != nil {
		return err
	}

	if err := validateKubermaticNotRunning(ctx, logger, opt); err != nil {
		return err
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
	logger.Info("Validating Validating that KKP is not running…")

	success := true

	// check master cluster
	if !validateKubermaticNotRunningInCluster(ctx, logger.WithField("cluster", "master"), opt.MasterClient, opt) {
		success = false
	}

	for seedName, seedClient := range opt.SeedClients {
		if !validateKubermaticNotRunningInCluster(ctx, logger.WithField("seed", seedName), seedClient, opt) {
			success = false
		}
	}

	if !success {
		return errors.New("please scale down all KKP deployments to 0 and remove KKP webhooks")
	}

	return nil
}

func validateKubermaticNotRunningInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, opt *Options) bool {
	logger.Info("Checking controllers…")

	success := true

	deployments := []string{
		"kubermatic-operator", // as named in our Helm chart
		common.MasterControllerManagerDeploymentName,
		common.SeedControllerManagerDeploymentName,
		"kubermatic-api", // TODO: make the constant for this public
	}

	for _, name := range deployments {
		if !validateDeploymentHasNoReplicas(ctx, logger, client, opt.KubermaticNamespace, name) {
			success = false
		}
	}

	logger.Info("Checking webhooks…")

	webhooks := []string{
		kubermatic.ClusterAdmissionWebhookName,
		common.SeedAdmissionWebhookName(opt.KubermaticConfiguration),
	}

	for _, name := range webhooks {
		if !validateWebhookDoesNotExist(ctx, logger, client, name) {
			success = false
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
