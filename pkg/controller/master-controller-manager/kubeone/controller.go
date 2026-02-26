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

package kubeone

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"time"

	"go.uber.org/zap"

	kubeonev1beta2 "k8c.io/kubeone/pkg/apis/kubeone/v1beta2"
	"k8c.io/kubeone/pkg/fail"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/machine-controller/sdk/providerconfig"
	reconcilerlog "k8c.io/reconciler/pkg/log"
	"k8c.io/reconciler/pkg/reconciling"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	reconcilerwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
)

const (
	// This controller is responsible for managing the lifecycle of KubeOne clusters within KKP.
	ControllerName = "kkp-kubeone-controller"

	JobNameLabel = "job-name"

	// kubernetes pod status.
	podPhaseKey = "status.phase"

	// ImportAction is the action to import kubeone cluster.
	ImportAction = "import"

	// UpgradeControlPlaneAction is the action to upgrade kubeone cluster.
	UpgradeControlPlaneAction = "upgrade"

	// MigrateContainerRuntimeAction is the action to migrate kubeone container-runtime.
	MigrateContainerRuntimeAction = "migrate"

	// KubeOneImportJob is the name of kubeone job performing import.
	KubeOneImportJob = "kubeone-import"

	// KubeOneUpgradeJob is the name of kubeone job performing upgrade.
	KubeOneUpgradeJob = "kubeone-upgrade"

	// Job back-off limit is set by default to 6.
	KubeOneJobBackOffLimit = 6

	// KubeOneMigrateJob is the name of kubeone job performing container-runtime migration.
	KubeOneMigrateJob = "kubeone-migrate"

	// KubeOneImportConfigMap is the name of kubeone configmap which stores import action script.
	KubeOneImportConfigMap = "kubeone-import"

	// KubeOneUpgradeConfigMap is the name of kubeone configmap which stores upgrade action script.
	KubeOneUpgradeConfigMap = "kubeone-upgrade"

	// KubeOneMigrateConfigMap is the name of kubeone configmap which stores migrate action script.
	KubeOneMigrateConfigMap = "kubeone-migrate"
)

type templateData interface {
	RewriteImage(image string) (string, error)
}

type reconciler struct {
	ctrlruntimeclient.Client
	log               *zap.SugaredLogger
	secretKeySelector provider.SecretKeySelectorValueFunc
	overwriteRegistry string
}

func Add(ctx context.Context, mgr manager.Manager, log *zap.SugaredLogger, overwriteRegistry string) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &corev1.Pod{}, podPhaseKey, func(rawObj ctrlruntimeclient.Object) []string {
		pod := rawObj.(*corev1.Pod)
		return []string{string(pod.Status.Phase)}
	}); err != nil {
		return fmt.Errorf("failed to add index on pod.Status.Phase: %w", err)
	}

	reconciler := &reconciler{
		Client:            mgr.GetClient(),
		log:               log.Named(ControllerName),
		secretKeySelector: provider.SecretKeySelectorValueFuncFactory(ctx, mgr.GetClient()),
		overwriteRegistry: overwriteRegistry,
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		For(&kubermaticv1.ExternalCluster{}, builder.WithPredicates(withEventFilter())).
		Owns(&batchv1.Job{}, builder.WithPredicates(updateEventsOnly())).
		Build(reconciler)

	return err
}

func updateEventsOnly() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectOld.GetResourceVersion() != e.ObjectNew.GetResourceVersion()
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func withEventFilter() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			externalCluster, ok := e.Object.(*kubermaticv1.ExternalCluster)
			if !ok {
				return false
			}
			if externalCluster.Spec.CloudSpec.ProviderName == "" {
				return false
			}
			return externalCluster.Spec.CloudSpec.KubeOne != nil
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	paused, err := kuberneteshelper.ExternalClusterPausedChecker(ctx, request.Name, r)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check kubeone external cluster pause status: %w", err)
	}
	if paused {
		return reconcile.Result{}, nil
	}

	log := r.log.With("externalcluster", request.Name)
	log.Debug("Processing")

	return reconcile.Result{}, r.reconcile(ctx, request.Name, log)
}

func (r *reconciler) reconcile(ctx context.Context, externalClusterName string, log *zap.SugaredLogger) error {
	externalCluster := &kubermaticv1.ExternalCluster{}
	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: metav1.NamespaceAll, Name: externalClusterName}, externalCluster); ctrlruntimeclient.IgnoreNotFound(err) != nil {
		return err
	}

	kubeOneSpec := externalCluster.Spec.CloudSpec.KubeOne
	if kubeOneSpec == nil {
		return nil
	}

	kubeOneNamespaceName := externalCluster.GetKubeOneNamespaceName()
	err := r.createKubeOneNamespace(ctx, kubeOneNamespaceName)
	if err != nil {
		return err
	}

	if !externalCluster.DeletionTimestamp.IsZero() {
		log.Info("Deleting KubeOne Namespace and Secrets...")
		kubeOneNamespace := &corev1.Namespace{}
		if err := r.Get(ctx, types.NamespacedName{Name: kubeOneNamespaceName}, kubeOneNamespace); err != nil {
			return err
		}
		kubeOneSecrests, err := r.getKubeOneSecrets(ctx, externalCluster)
		if err != nil {
			return err
		}
		if err := r.handleDeletion(ctx, log, kubeOneNamespace, externalCluster, kubeOneSecrests); err != nil {
			return fmt.Errorf("failed deleting kubeone externalcluster: %w", err)
		}
		return nil
	}

	data := resources.NewTemplateDataBuilder().
		WithOverwriteRegistry(r.overwriteRegistry).
		Build()

	kubeOneSecrests, err := r.ensureKubeOneSecrets(ctx, log, data, externalCluster)
	if err != nil {
		return err
	}
	if err := r.syncSecrets(ctx, externalCluster, kubeOneSecrests); err != nil {
		return err
	}

	KubeOneFinalizerList := []string{
		kubermaticv1.ExternalClusterKubeOneNamespaceCleanupFinalizer,
		kubermaticv1.ExternalClusterKubeOneSecretsCleanupFinalizer,
	}
	finalizersToAdd := sets.NewString(KubeOneFinalizerList...).Difference(sets.NewString(externalCluster.GetFinalizers()...))
	finalizersToAddList := finalizersToAdd.UnsortedList()

	if len(finalizersToAddList) > 0 {
		if err := kuberneteshelper.TryAddFinalizer(ctx, r, externalCluster, finalizersToAddList...); err != nil {
			return fmt.Errorf("failed to add kubeone namespace finalizer: %w", err)
		}
	}

	if err := r.importAction(ctx, log, data, externalCluster); err != nil {
		return err
	}

	if err = r.upgradeAction(ctx, log, data, externalCluster); err != nil {
		return err
	}

	if err = r.migrateAction(ctx, log, data, externalCluster); err != nil {
		return err
	}

	return nil
}

func (r *reconciler) syncSecrets(ctx context.Context, externalCluster *kubermaticv1.ExternalCluster, kubeOneSecrests []corev1.Secret) error {
	for _, secret := range kubeOneSecrests {
		if _, err := kubernetesprovider.CreateOrUpdateSecretForCluster(ctx, r, externalCluster, secret.Data, secret.Name, externalCluster.GetKubeOneNamespaceName()); ctrlruntimeclient.IgnoreAlreadyExists(err) != nil {
			return err
		}
	}

	return nil
}

func (r *reconciler) ensureKubeOneSecrets(
	ctx context.Context,
	log *zap.SugaredLogger,
	data templateData,
	externalCluster *kubermaticv1.ExternalCluster,
) ([]corev1.Secret, error) {
	kubeOneSecrets := []corev1.Secret{}

	credRef := externalCluster.Spec.CloudSpec.KubeOne.CredentialsReference
	if credRef != nil {
		credSecret := &corev1.Secret{}
		err := r.Get(ctx, types.NamespacedName{Name: credRef.Name, Namespace: credRef.Namespace}, credSecret)
		if err != nil {
			if apierrors.IsNotFound(err) {
				if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterCondition{
					Phase:   kubermaticv1.ExternalClusterPhaseError,
					Message: fmt.Sprintf("missing credential secret for cluster %s but credential secret reference exists in cluster. Please create credential secret for cluster.", externalCluster.Name),
				}); err != nil {
					return nil, err
				}
			}
			return nil, err
		}
		kubeOneSecrets = append(kubeOneSecrets, *credSecret)
	} else {
		if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterCondition{
			Phase:   kubermaticv1.ExternalClusterPhaseError,
			Message: fmt.Sprintf("missing credential secret reference for cluster %s. Please add credential secret reference to this cluster object.", externalCluster.Name),
		}); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("missing credential secret reference for cluster %s.", externalCluster.Name)
	}

	manifestRef := externalCluster.Spec.CloudSpec.KubeOne.ManifestReference
	if manifestRef != nil {
		manifestSecret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: manifestRef.Name, Namespace: manifestRef.Namespace}, manifestSecret); err != nil {
			if apierrors.IsNotFound(err) {
				if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterCondition{
					Phase:   kubermaticv1.ExternalClusterPhaseError,
					Message: fmt.Sprintf("missing manifest secret for cluster %s but manifest secret reference exists in cluster. Please create manifest secret for cluster.", externalCluster.Name),
				}); err != nil {
					return nil, err
				}
			}
			return nil, err
		}
		kubeOneSecrets = append(kubeOneSecrets, *manifestSecret)
	} else {
		if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterCondition{
			Phase:   kubermaticv1.ExternalClusterPhaseError,
			Message: fmt.Sprintf("missing manifest secret reference for cluster %s. Please add manifest secret reference to this cluster object.", externalCluster.Name),
		}); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("missing manifest secret reference for cluster %s", externalCluster.Name)
	}

	sshRef := externalCluster.Spec.CloudSpec.KubeOne.SSHReference
	if sshRef != nil {
		sshSecret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: sshRef.Name, Namespace: sshRef.Namespace}, sshSecret); err != nil {
			if apierrors.IsNotFound(err) {
				if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterCondition{
					Phase:   kubermaticv1.ExternalClusterPhaseError,
					Message: fmt.Sprintf("missing ssh secret for cluster %s but ssh secret reference exists in cluster. Please create ssh secret for cluster.", externalCluster.Name),
				}); err != nil {
					return nil, err
				}
			}
			return nil, err
		}
		kubeOneSecrets = append(kubeOneSecrets, *sshSecret)
	} else {
		if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterCondition{
			Phase:   kubermaticv1.ExternalClusterPhaseError,
			Message: fmt.Sprintf("missing ssh secret reference for cluster %s. Please add ssh secret reference to this cluster object.", externalCluster.Name),
		}); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("missing ssh secret reference for cluster %s", externalCluster.Name)
	}

	kubeconfigRef := externalCluster.Spec.KubeconfigReference
	// kubeconfigRef can be nil when cluster is in Provisioning phase
	if kubeconfigRef != nil {
		kubeconfigsecret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: kubeconfigRef.Name, Namespace: kubeconfigRef.Namespace}, kubeconfigsecret); err != nil {
			if apierrors.IsNotFound(err) {
				// trying to refetch cluster kubeconfig to recreate kubeconfig secret in case kubeconfig secret was deleted for some reason.
				log.Info("trying to refetch cluster kubeconfig to recreate kubeconfig secret in case kubeconfig secret was deleted for some reason.")
				err := r.initiateImportCluster(ctx, log, data, externalCluster)
				if err != nil {
					log.Errorw("failed to import kubeone cluster", zap.Error(err))
					return nil, err
				}
			} else {
				return nil, err
			}
		}
		kubeOneSecrets = append(kubeOneSecrets, *kubeconfigsecret)
	}

	return kubeOneSecrets, nil
}

func (r *reconciler) getKubeOneSecrets(ctx context.Context, externalCluster *kubermaticv1.ExternalCluster) ([]corev1.Secret, error) {
	kubeOneSecrests := []corev1.Secret{}

	credRef := externalCluster.Spec.CloudSpec.KubeOne.CredentialsReference
	if credRef != nil {
		credSecret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: credRef.Name, Namespace: credRef.Namespace}, credSecret); ctrlruntimeclient.IgnoreNotFound(err) != nil {
			return nil, err
		}
		kubeOneSecrests = append(kubeOneSecrests, *credSecret)
	}

	manifestRef := externalCluster.Spec.CloudSpec.KubeOne.ManifestReference
	if manifestRef != nil {
		manifestSecret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: manifestRef.Name, Namespace: manifestRef.Namespace}, manifestSecret); ctrlruntimeclient.IgnoreNotFound(err) != nil {
			return nil, err
		}
		kubeOneSecrests = append(kubeOneSecrests, *manifestSecret)
	}

	sshRef := externalCluster.Spec.CloudSpec.KubeOne.SSHReference
	if sshRef != nil {
		sshSecret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: sshRef.Name, Namespace: sshRef.Namespace}, sshSecret); ctrlruntimeclient.IgnoreNotFound(err) != nil {
			return nil, err
		}
		kubeOneSecrests = append(kubeOneSecrests, *sshSecret)
	}

	kubeconfigRef := externalCluster.Spec.KubeconfigReference
	if kubeconfigRef != nil {
		kubeconfigSecret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: kubeconfigRef.Name, Namespace: kubeconfigRef.Namespace}, kubeconfigSecret); ctrlruntimeclient.IgnoreNotFound(err) != nil {
			return nil, err
		}
		kubeOneSecrests = append(kubeOneSecrests, *kubeconfigSecret)
	}

	return kubeOneSecrests, nil
}

func (r *reconciler) deleteSecrets(ctx context.Context, secrets []corev1.Secret) error {
	for _, secret := range secrets {
		if err := r.Delete(ctx, &secret); ctrlruntimeclient.IgnoreNotFound(err) != nil {
			return err
		}
	}

	return nil
}

func (r *reconciler) importAction(
	ctx context.Context,
	log *zap.SugaredLogger,
	data templateData,
	externalCluster *kubermaticv1.ExternalCluster,
) error {
	if externalCluster.Spec.KubeconfigReference == nil {
		err := r.initiateImportCluster(ctx, log, data, externalCluster)
		if err != nil {
			log.Errorw("failed to import kubeone cluster", zap.Error(err))
			return err
		}
	} else if externalCluster.Spec.KubeconfigReference != nil && externalCluster.Status.Condition.Phase == kubermaticv1.ExternalClusterPhaseRunning {
		// checking if cluster is accessible using client
		clusterClient, err := kuberneteshelper.GetClusterClient(ctx, externalCluster, r)
		if err != nil {
			return err
		}
		_, err = kuberneteshelper.GetVersion(clusterClient)
		if err != nil {
			if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterCondition{
				Phase:   kubermaticv1.ExternalClusterPhaseError,
				Message: err.Error(),
			}); err != nil {
				return err
			}
			return err
		}
	}

	return nil
}

func (r *reconciler) initiateImportCluster(
	ctx context.Context,
	log *zap.SugaredLogger,
	data templateData,
	externalCluster *kubermaticv1.ExternalCluster,
) error {
	log.Info("Importing kubeone cluster...")

	kubeoneNamespace := externalCluster.GetKubeOneNamespaceName()

	if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterCondition{
		Phase:   kubermaticv1.ExternalClusterPhaseProvisioning,
		Message: fmt.Sprintf("trying to fetch cluster %s kubeconfig", externalCluster.Name),
	}); err != nil {
		return err
	}

	log.Info("Generating kubeone job to fetch kubeconfig...")
	job, err := r.generateKubeOneActionJob(ctx, log, data, externalCluster, ImportAction)
	if err != nil {
		return fmt.Errorf("could not generate kubeone job: %w", err)
	}

	log.Info("Creating kubeone job to fetch kubeconfig...")
	if err := r.Create(ctx, job); ctrlruntimeclient.IgnoreAlreadyExists(err) != nil {
		return fmt.Errorf("could not create kubeone job %s/%s: %w", job.Name, job.Namespace, err)
	}

	// Wait until the object exists in the cache
	namespacedName := types.NamespacedName{Name: job.Name, Namespace: job.Namespace}
	createdObjectIsInCache := reconciling.WaitUntilObjectExistsInCacheConditionFunc(r, objectLogger(job), namespacedName, job)
	err = reconcilerwait.PollUntilContextTimeout(ctx, 10*time.Millisecond, 10*time.Second, true, createdObjectIsInCache)
	if err != nil {
		return fmt.Errorf("failed waiting for the cache to contain our newly created object: %w", err)
	}

	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: job.Namespace, Name: job.Name}, job); err != nil {
		return fmt.Errorf("failed to get kubeone kubeconfig job: %w", err)
	}

	// job failed.
	if job.Status.Failed > KubeOneJobBackOffLimit {
		log.Info("Kubeone import failed!")
		// update kubeone externalcluster status.
		if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterCondition{
			Phase:   kubermaticv1.ExternalClusterPhaseError,
			Message: fmt.Sprintf("kubeone cluster %s import failed", externalCluster.Name),
		}); err != nil {
			return err
		}
		// delete kubeone job alongwith its pods as no longer required.
		propagationPolicy := metav1.DeletePropagationBackground
		err = r.Delete(ctx, job, &ctrlruntimeclient.DeleteOptions{PropagationPolicy: &propagationPolicy})
		if ctrlruntimeclient.IgnoreNotFound(err) != nil {
			return err
		}
		return errors.New("kubeone import failed")
	}

	podList := &corev1.PodList{}
	err = r.List(ctx,
		podList,
		&ctrlruntimeclient.ListOptions{
			Limit:         1,
			Namespace:     job.Namespace,
			FieldSelector: fields.OneTermEqualSelector(podPhaseKey, string(corev1.PodSucceeded)),
		},
		&ctrlruntimeclient.MatchingLabels{JobNameLabel: job.Name},
	)
	if err != nil {
		return err
	}

	if len(podList.Items) == 0 {
		return fmt.Errorf("no succeeded import pods in kubeone namespace:%s", kubeoneNamespace)
	}

	succeededPod := podList.Items[0]
	// use pod logs to get cluster's kubeconfig.
	config, err := getPodLogs(ctx, &succeededPod)
	if err != nil {
		return err
	}

	// verify kubeconfig
	kubeconfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(config))
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return err
	}
	_, err = clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		return fmt.Errorf("kubeone cluster kubeconfig not valid: %w", err)
	}

	log.Info("Fetched config from the kubeone pod")

	secretName := externalCluster.GetKubeconfigSecretName()
	secretNamespace := resources.KubermaticNamespace
	secretData := map[string][]byte{
		resources.ExternalClusterKubeconfig: []byte(config),
	}
	kubeconfigRef, err := kubernetesprovider.CreateOrUpdateSecretForCluster(ctx, r, externalCluster, secretData, secretName, secretNamespace)
	if err != nil {
		return err
	}

	oldexternalCluster := externalCluster.DeepCopy()
	externalCluster.Spec.KubeconfigReference = kubeconfigRef
	if err := r.Patch(ctx, externalCluster, ctrlruntimeclient.MergeFrom(oldexternalCluster)); err != nil {
		log.Errorw("failed to add kubeconfig reference in external cluster", zap.Error(err))
		return err
	}

	log.Info("KubeOne Cluster Imported!")
	if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterCondition{
		Phase: kubermaticv1.ExternalClusterPhaseRunning,
	}); err != nil {
		return err
	}
	// delete kubeone job alongwith its pods as no longer required.
	propagationPolicy := metav1.DeletePropagationBackground
	err = r.Delete(ctx, job, &ctrlruntimeclient.DeleteOptions{PropagationPolicy: &propagationPolicy})
	return ctrlruntimeclient.IgnoreNotFound(err)
}

func (r *reconciler) upgradeAction(
	ctx context.Context,
	log *zap.SugaredLogger,
	data templateData,
	externalCluster *kubermaticv1.ExternalCluster,
) error {
	manifestRef := externalCluster.Spec.CloudSpec.KubeOne.ManifestReference
	kubeOneNamespaceName := externalCluster.GetKubeOneNamespaceName()

	clusterClient, err := kuberneteshelper.GetClusterClient(ctx, externalCluster, r)
	if err != nil {
		return err
	}

	currentVersion, err := kuberneteshelper.GetVersion(clusterClient)
	if err != nil {
		return err
	}
	desiredVersion := externalCluster.Spec.Version

	desiredPhases := []string{
		string(kubermaticv1.KubeOnePhaseReconcilingUpgrade),
		string(kubermaticv1.ExternalClusterPhaseError),
		string(kubermaticv1.ExternalClusterPhaseRunning),
	}
	desiredPhaseBool := sets.NewString(desiredPhases...).Has(string(externalCluster.Status.Condition.Phase))

	// check if pod succeeded
	podList := &corev1.PodList{}
	err = r.List(ctx,
		podList,
		&ctrlruntimeclient.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(podPhaseKey, string(corev1.PodSucceeded)),
			Namespace:     kubeOneNamespaceName,
			Limit:         1,
		},
		&ctrlruntimeclient.MatchingLabels{JobNameLabel: KubeOneUpgradeJob},
	)
	if err != nil {
		return err
	}
	if externalCluster.Status.Condition.Phase == kubermaticv1.KubeOnePhaseReconcilingUpgrade && len(podList.Items) == 1 && desiredVersion.Equal(currentVersion) {
		log.Info("KubeOne Cluster Upgraded!")
		if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterCondition{
			Phase: kubermaticv1.ExternalClusterPhaseRunning,
		}); err != nil {
			return err
		}
		// delete kubeone job alongwith its pods as no longer required.
		propagationPolicy := metav1.DeletePropagationBackground

		err := r.Delete(ctx, &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      KubeOneUpgradeJob,
				Namespace: externalCluster.GetKubeOneNamespaceName(),
			},
		}, &ctrlruntimeclient.DeleteOptions{PropagationPolicy: &propagationPolicy})

		return ctrlruntimeclient.IgnoreNotFound(err)
	}

	// proceed only when desiredVersion is grater from currentVersion or clusterphase is desiredPhase.
	if desiredVersion.Equal(currentVersion) || desiredVersion.LessThan(currentVersion) || !desiredPhaseBool {
		return nil
	}

	log.Infow("Upgrading kubeone cluster...", "from", currentVersion, "to", desiredVersion)

	// Update KubeOne Manifest
	manifestSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: manifestRef.Namespace, Name: manifestRef.Name}, manifestSecret); err != nil {
		return err
	}
	currentManifest := manifestSecret.Data[resources.KubeOneManifest]

	kubeOneClusterObj := &kubeonev1beta2.KubeOneCluster{}
	if err := yaml.UnmarshalStrict(currentManifest, kubeOneClusterObj); err != nil {
		return fmt.Errorf("failed to decode kubeone manifest secret data: %w", err)
	}
	kubeOneClusterObj.Versions = kubeonev1beta2.VersionConfig{
		Kubernetes: desiredVersion.String(),
	}

	patchManifest, err := yaml.Marshal(kubeOneClusterObj)
	if err != nil {
		return fmt.Errorf("failed to encode kubeone cluster manifest config as YAML: %w", err)
	}

	oldManifestSecret := manifestSecret.DeepCopy()
	manifestSecret.Data = map[string][]byte{
		resources.KubeOneManifest: patchManifest,
	}
	if err := r.Patch(ctx, manifestSecret, ctrlruntimeclient.MergeFrom(oldManifestSecret)); err != nil {
		return fmt.Errorf("failed to update kubeone manifest secret for upgrade version %s/%s: %w", manifestSecret.Name, manifestSecret.Namespace, err)
	}
	if _, err := kubernetesprovider.CreateOrUpdateSecretForCluster(ctx, r, externalCluster, manifestSecret.Data, manifestSecret.Name, externalCluster.GetKubeOneNamespaceName()); ctrlruntimeclient.IgnoreAlreadyExists(err) != nil {
		return err
	}

	if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterCondition{
		Phase:   kubermaticv1.KubeOnePhaseReconcilingUpgrade,
		Message: fmt.Sprintf("upgrading cluster %v version from %v to %v", externalCluster, currentVersion, desiredVersion),
	}); err != nil {
		return err
	}

	err = r.initiateClusterUpgrade(ctx, log, data, *currentVersion, desiredVersion, externalCluster)
	if err != nil {
		log.Errorw("failed to upgrade kubeone cluster", zap.Error(err))
		return err
	}

	return nil
}

func (r *reconciler) initiateClusterUpgrade(
	ctx context.Context,
	log *zap.SugaredLogger,
	data templateData,
	currentVersion semver.Semver,
	desiredVersion semver.Semver,
	cluster *kubermaticv1.ExternalCluster,
) error {
	log.Info("Upgrading kubeone cluster...")

	job, err := r.generateKubeOneActionJob(ctx, log, data, cluster, UpgradeControlPlaneAction)
	if err != nil {
		return err
	}

	if err := r.Create(ctx, job); ctrlruntimeclient.IgnoreAlreadyExists(err) != nil {
		return err
	}

	// Wait until the object exists in the cache
	namespacedName := types.NamespacedName{Name: job.Name, Namespace: job.Namespace}
	createdObjectIsInCache := reconciling.WaitUntilObjectExistsInCacheConditionFunc(r, objectLogger(job), namespacedName, job)
	err = reconcilerwait.PollUntilContextTimeout(ctx, 10*time.Millisecond, 10*time.Second, true, createdObjectIsInCache)
	if err != nil {
		return fmt.Errorf("failed waiting for the cache to contain our newly created object: %w", err)
	}

	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: job.Namespace, Name: job.Name}, job); err != nil {
		return fmt.Errorf("failed to get kubeone upgrade job: %w", err)
	}

	if job.Status.Failed > KubeOneJobBackOffLimit {
		log.Info("Kubeone upgrade failed!")
		if err := r.updateClusterStatus(ctx, cluster, kubermaticv1.ExternalClusterCondition{
			Phase:   kubermaticv1.ExternalClusterPhaseError,
			Message: fmt.Sprintf("cluster %s upgrade failed", cluster.Name),
		}); err != nil {
			return err
		}
		// delete kubeone job alongwith its pods as no longer required.
		propagationPolicy := metav1.DeletePropagationBackground
		err = r.Delete(ctx, job, &ctrlruntimeclient.DeleteOptions{PropagationPolicy: &propagationPolicy})
		if ctrlruntimeclient.IgnoreNotFound(err) != nil {
			return err
		}
		return errors.New("kubeone upgrade failed")
	}

	return nil
}

func objectLogger(obj ctrlruntimeclient.Object) *zap.SugaredLogger {
	// make sure we handle objects with broken typeMeta and still create a nice-looking kind name
	logger := reconcilerlog.Logger().With("kind", reflect.TypeOf(obj).Elem())
	if ns := obj.GetNamespace(); ns != "" {
		logger = logger.With("namespace", ns)
	}

	// ensure name comes after namespace
	return logger.With("name", obj.GetName())
}

func (r *reconciler) migrateAction(
	ctx context.Context,
	log *zap.SugaredLogger,
	data templateData,
	externalCluster *kubermaticv1.ExternalCluster,
) error {
	manifestRef := externalCluster.Spec.CloudSpec.KubeOne.ManifestReference

	clusterClient, err := kuberneteshelper.GetClusterClient(ctx, externalCluster, r)
	if err != nil {
		return err
	}
	currentContainerRuntime, err := kuberneteshelper.GetContainerRuntime(ctx, clusterClient)
	if err != nil {
		return err
	}
	desiredContainerRuntime := externalCluster.Spec.ContainerRuntime

	// reached desired state
	if currentContainerRuntime == desiredContainerRuntime && externalCluster.Status.Condition.Phase == kubermaticv1.KubeOnePhaseReconcilingMigrate {
		log.Info("KubeOne Cluster Migrated!")
		if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterCondition{
			Phase: kubermaticv1.ExternalClusterPhaseRunning,
		}); err != nil {
			return err
		}
		// delete kubeone job alongwith its pods as no longer required.
		propagationPolicy := metav1.DeletePropagationBackground

		err := r.Delete(ctx, &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      KubeOneMigrateJob,
				Namespace: externalCluster.GetKubeOneNamespaceName(),
			},
		}, &ctrlruntimeclient.DeleteOptions{PropagationPolicy: &propagationPolicy})

		return ctrlruntimeclient.IgnoreNotFound(err)
	}

	desiredPhases := []string{
		string(kubermaticv1.KubeOnePhaseReconcilingMigrate),
		string(kubermaticv1.ExternalClusterPhaseError),
		string(kubermaticv1.ExternalClusterPhaseRunning),
	}
	desiredPhaseBool := sets.NewString(desiredPhases...).Has(string(externalCluster.Status.Condition.Phase))
	if !desiredPhaseBool || currentContainerRuntime == desiredContainerRuntime || desiredContainerRuntime != resources.ContainerRuntimeContainerd {
		return nil
	}

	log.Infow("Migrating kubeone cluster container runtime...", "from", currentContainerRuntime, "to", desiredContainerRuntime)

	// Update KubeOne Manifest
	manifestSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: manifestRef.Namespace, Name: manifestRef.Name}, manifestSecret); err != nil {
		return err
	}
	currentManifest := manifestSecret.Data[resources.KubeOneManifest]

	kubeOneClusterObj := &kubeonev1beta2.KubeOneCluster{}
	if err := yaml.UnmarshalStrict(currentManifest, kubeOneClusterObj); err != nil {
		return fmt.Errorf("failed to decode kubeone manifest secret data: %w", err)
	}
	if kubeOneClusterObj.ContainerRuntime.Docker != nil {
		kubeOneClusterObj.ContainerRuntime.Docker = nil
	}
	kubeOneClusterObj.ContainerRuntime.Containerd = &kubeonev1beta2.ContainerRuntimeContainerd{}

	patchManifest, err := yaml.Marshal(kubeOneClusterObj)
	if err != nil {
		return fmt.Errorf("failed to encode kubeone cluster manifest config as YAML: %w", err)
	}

	oldManifestSecret := manifestSecret.DeepCopy()
	manifestSecret.Data = map[string][]byte{
		resources.KubeOneManifest: patchManifest,
	}
	if err := r.Patch(ctx, manifestSecret, ctrlruntimeclient.MergeFrom(oldManifestSecret)); err != nil {
		return fmt.Errorf("failed to update kubeone manifest secret for upgrade version %s/%s: %w", manifestSecret.Name, manifestSecret.Namespace, err)
	}
	if _, err := kubernetesprovider.CreateOrUpdateSecretForCluster(ctx, r, externalCluster, manifestSecret.Data, manifestSecret.Name, externalCluster.GetKubeOneNamespaceName()); ctrlruntimeclient.IgnoreAlreadyExists(err) != nil {
		return err
	}

	err = r.initiateClusterMigration(ctx, log, data, currentContainerRuntime, desiredContainerRuntime, externalCluster)
	if err != nil {
		log.Errorw("failed to migrate kubeone cluster", zap.Error(err))
		return err
	}

	return nil
}

func (r *reconciler) initiateClusterMigration(
	ctx context.Context,
	log *zap.SugaredLogger,
	data templateData,
	currentContainerRuntime string,
	desiredContainerRuntime string,
	cluster *kubermaticv1.ExternalCluster,
) error {
	log.Info("Migrating kubeone cluster...")
	if err := r.updateClusterStatus(ctx, cluster, kubermaticv1.ExternalClusterCondition{
		Phase:   kubermaticv1.KubeOnePhaseReconcilingMigrate,
		Message: fmt.Sprintf("migrating cluster %s container runtime from %v to %v", cluster.Name, currentContainerRuntime, desiredContainerRuntime),
	}); err != nil {
		return err
	}

	job, err := r.generateKubeOneActionJob(ctx, log, data, cluster, MigrateContainerRuntimeAction)
	if err != nil {
		return fmt.Errorf("could not generate kubeone pod %s/%s to migrate container runtime: %w", job.Name, job.Namespace, err)
	}

	log.Info("Creating kubeone job to migrate kubeone...")
	if err := r.Create(ctx, job); ctrlruntimeclient.IgnoreAlreadyExists(err) != nil {
		return fmt.Errorf("could not create kubeone job %s/%s to migrate kubeone cluster: %w", job.Name, job.Namespace, err)
	}

	// Wait until the object exists in the cache
	namespacedName := types.NamespacedName{Name: job.Name, Namespace: job.Namespace}
	createdObjectIsInCache := reconciling.WaitUntilObjectExistsInCacheConditionFunc(r, objectLogger(job), namespacedName, job)
	err = reconcilerwait.PollUntilContextTimeout(ctx, 10*time.Millisecond, 10*time.Second, true, createdObjectIsInCache)
	if err != nil {
		return fmt.Errorf("failed waiting for the cache to contain our newly created object: %w", err)
	}

	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: job.Namespace, Name: job.Name}, job); err != nil {
		return fmt.Errorf("failed to get kubeone migrate job: %w", err)
	}

	// job failed.
	if job.Status.Failed > KubeOneJobBackOffLimit {
		log.Info("Kubeone migration failed!")
		if err := r.updateClusterStatus(ctx, cluster, kubermaticv1.ExternalClusterCondition{
			Phase:   kubermaticv1.ExternalClusterPhaseError,
			Message: fmt.Sprintf("cluster %s upgrade failed from %v to %v", cluster.Name, currentContainerRuntime, desiredContainerRuntime),
		}); err != nil {
			return err
		}
		// delete kubeone job alongwith its pods as no longer required.
		propagationPolicy := metav1.DeletePropagationBackground
		err = r.Delete(ctx, job, &ctrlruntimeclient.DeleteOptions{PropagationPolicy: &propagationPolicy})
		if ctrlruntimeclient.IgnoreNotFound(err) != nil {
			return err
		}
		return errors.New("kubeone migration failed")
	}

	return nil
}

func (r *reconciler) generateKubeOneActionJob(ctx context.Context, log *zap.SugaredLogger, data templateData, externalCluster *kubermaticv1.ExternalCluster, action string) (*batchv1.Job, error) {
	var kubeoneJobName, kubeoneCMName string
	var sshSecret, manifestSecret *corev1.Secret
	var err error
	kubeOne := externalCluster.Spec.CloudSpec.KubeOne
	envVar := []corev1.EnvVar{}
	volumes := []corev1.Volume{}
	providerName := kubeOne.ProviderName
	kubeOneNamespaceName := externalCluster.GetKubeOneNamespaceName()

	sshSecret, err = r.getKubeOneSecret(ctx, *kubeOne.SSHReference)
	if err != nil {
		log.Errorw("could not find kubeone ssh secret", zap.Error(err))
		return nil, err
	}

	manifestSecret, err = r.getKubeOneSecret(ctx, *kubeOne.ManifestReference)
	if err != nil {
		log.Errorw("could not find kubeone manifest secret", zap.Error(err))
		return nil, err
	}

	// provider credentials not required for importing the cluster.
	if action != ImportAction {
		credentialSecret, err := r.getKubeOneSecret(ctx, *kubeOne.CredentialsReference)
		if err != nil {
			log.Errorw("could not find kubeone credential secret", zap.Error(err))
			return nil, err
		}
		envVar = setEnvForProvider(providerName, envVar, credentialSecret)
	}

	// storing kubeone pod scripts in a configMap.
	cm := generateConfigMap(kubeOneNamespaceName, action)
	if err := r.Create(ctx, cm); ctrlruntimeclient.IgnoreAlreadyExists(err) != nil {
		return nil, fmt.Errorf("failed to create kubeone script configmap: %w", err)
	}

	_, ok := sshSecret.Data[resources.KubeOneSSHPassphrase]
	if ok {
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name: "PASSPHRASE",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: sshSecret.Name,
						},
						Key: resources.KubeOneSSHPassphrase,
					},
				},
			},
		)
	}

	vm := []corev1.VolumeMount{}
	vmInit := []corev1.VolumeMount{}

	vmInit = append(
		vmInit,
		corev1.VolumeMount{
			Name:      "rw-manifest-volume",
			MountPath: "/kubeonemanifest",
		},
		corev1.VolumeMount{
			Name:      "manifest-volume",
			MountPath: "/manifest",
		},
	)

	vm = append(
		vm,
		corev1.VolumeMount{
			Name:      "ssh-volume",
			MountPath: "/root/.ssh",
		},
		corev1.VolumeMount{
			Name:      "rw-manifest-volume",
			MountPath: "/kubeonemanifest",
		},
		corev1.VolumeMount{
			Name:      "script-volume",
			MountPath: "/scripts",
		},
	)

	switch action {
	case ImportAction:
		kubeoneJobName = KubeOneImportJob
		kubeoneCMName = KubeOneImportConfigMap
	case UpgradeControlPlaneAction:
		kubeoneJobName = KubeOneUpgradeJob
		kubeoneCMName = KubeOneUpgradeConfigMap
	case MigrateContainerRuntimeAction:
		kubeoneJobName = KubeOneMigrateJob
		kubeoneCMName = KubeOneMigrateConfigMap
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubeoneJobName,
			Namespace: kubeOneNamespaceName,
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       externalCluster.Name,
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.ExternalClusterKind,
					Controller: ptr.To(true),
					UID:        externalCluster.GetUID(),
				},
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: ptr.To[int32](KubeOneJobBackOffLimit),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							Name:       externalCluster.Name,
							APIVersion: kubermaticv1.SchemeGroupVersion.String(),
							Kind:       kubermaticv1.ExternalClusterKind,
							Controller: ptr.To(true),
							UID:        externalCluster.GetUID(),
						},
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:    "copy-ro-manifest",
							Image:   registry.Must(data.RewriteImage("registry.k8s.io/e2e-test-images/busybox:1.29-2")),
							Command: []string{"/bin/sh"},
							Args: []string{
								"-c",
								"cp /manifest/* /kubeonemanifest",
							},
							VolumeMounts: vmInit,
						},
					},
					Containers: []corev1.Container{
						{
							Name:    "kubeone",
							Image:   registry.Must(data.RewriteImage(fmt.Sprintf("%s:%s", resources.KubeOneImage, resources.KubeOneImageTag))),
							Command: []string{"/bin/sh"},
							Args: []string{
								"-c",
								"./scripts/script.sh",
							},
							Env:          envVar,
							Resources:    corev1.ResourceRequirements{},
							VolumeMounts: vm,
						},
					},
					Volumes: append(
						volumes,
						corev1.Volume{
							Name: "rw-manifest-volume",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						corev1.Volume{
							Name: "manifest-volume",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: manifestSecret.Name,
								},
							},
						},
						corev1.Volume{
							Name: "ssh-volume",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									DefaultMode: ptr.To[int32](256),
									SecretName:  sshSecret.Name,
								},
							},
						},
						corev1.Volume{
							Name: "script-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: kubeoneCMName,
									},
									DefaultMode: ptr.To[int32](448),
								},
							},
						},
					),
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	return job, nil
}

func setEnvForProvider(providerName string, envVar []corev1.EnvVar, credentialSecret *corev1.Secret) []corev1.EnvVar {
	envVarSource := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: credentialSecret.Name,
			},
		},
	}

	if providerName == resources.KubeOneAWS {
		envVarSource.SecretKeyRef.Key = resources.AWSAccessKeyID
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "AWS_ACCESS_KEY_ID",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.AWSSecretAccessKey
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "AWS_SECRET_ACCESS_KEY",
				ValueFrom: envVarSource,
			},
		)
	}
	if providerName == resources.KubeOneAzure {
		envVarSource.SecretKeyRef.Key = resources.AzureClientID
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "ARM_CLIENT_ID",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.AzureClientSecret
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "ARM_CLIENT_SECRET",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.AzureTenantID
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "ARM_TENANT_ID",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.AzureSubscriptionID
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "ARM_SUBSCRIPTION_ID",
				ValueFrom: envVarSource,
			},
		)
	}
	if providerName == resources.KubeOneDigitalOcean {
		envVarSource.SecretKeyRef.Key = resources.DigitaloceanToken
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "DIGITALOCEAN_TOKEN",
				ValueFrom: envVarSource,
			},
		)
	}
	if providerName == resources.KubeOneGCP {
		envVarSource.SecretKeyRef.Key = resources.GCPServiceAccount
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "GOOGLE_CREDENTIALS",
				ValueFrom: envVarSource,
			},
		)
	}
	if providerName == resources.KubeOneHetzner {
		envVarSource.SecretKeyRef.Key = resources.HetznerToken
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "HCLOUD_TOKEN",
				ValueFrom: envVarSource,
			},
		)
	}
	if providerName == resources.KubeOneNutanix {
		envVarSource.SecretKeyRef.Key = resources.NutanixEndpoint
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "NUTANIX_ENDPOINT",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.NutanixPort
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "NUTANIX_PORT",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.NutanixUsername
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "NUTANIX_USERNAME",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.NutanixPassword
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "NUTANIX_PASSWORD",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.NutanixCSIEndpoint
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "NUTANIX_PE_ENDPOINT",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.NutanixCSIUsername
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "NUTANIX_PE_USERNAME",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.NutanixCSIPassword
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "NUTANIX_PE_PASSWORD",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.NutanixAllowInsecure
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "NUTANIX_INSECURE",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.NutanixProxyURL
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "NUTANIX_PROXY_URL",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.NutanixClusterName
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "NUTANIX_CLUSTER_NAME",
				ValueFrom: envVarSource,
			},
		)
	}
	if providerName == resources.KubeOneOpenStack {
		envVarSource.SecretKeyRef.Key = resources.OpenstackAuthURL
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "OS_AUTH_URL",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.OpenstackUsername
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "OS_USERNAME",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.OpenstackPassword
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "OS_PASSWORD",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.OpenstackRegion
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "OS_REGION_NAME",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.OpenstackDomain
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "OS_DOMAIN_NAME",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.OpenstackTenantID
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "OS_TENANT_ID",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.OpenstackTenant
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "OS_TENANT_NAME",
				ValueFrom: envVarSource,
			},
		)
	}
	if providerName == resources.KubeOneVSphere {
		envVarSource.SecretKeyRef.Key = resources.VsphereServer
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "VSPHERE_SERVER",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.VsphereUsername
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "VSPHERE_USER",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.VspherePassword
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "VSPHERE_PASSWORD",
				ValueFrom: envVarSource,
			},
		)
	}

	return envVar
}

func generateConfigMap(namespace, action string) *corev1.ConfigMap {
	var name, scriptToRun string
	scriptToRun = resources.KubeOneScript

	switch action {
	case ImportAction:
		name = KubeOneImportConfigMap
		scriptToRun += "kubeone kubeconfig --manifest kubeonemanifest/manifest 2> /dev/null"
	case UpgradeControlPlaneAction:
		name = KubeOneUpgradeConfigMap
		scriptToRun += "kubeone apply --manifest kubeonemanifest/manifest -y --log-format json"
	case MigrateContainerRuntimeAction:
		name = KubeOneMigrateConfigMap
		scriptToRun += "kubeone migrate to-containerd --manifest kubeonemanifest/manifest --log-format json"
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			"script.sh": scriptToRun,
		},
	}
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, ns *corev1.Namespace, externalCluster *kubermaticv1.ExternalCluster, kubeoneSecrets []corev1.Secret) error {
	if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterCondition{
		Phase: kubermaticv1.ExternalClusterPhaseDeleting,
	}); err != nil {
		return err
	}
	if kuberneteshelper.HasFinalizer(externalCluster, kubermaticv1.ExternalClusterKubeOneNamespaceCleanupFinalizer) {
		if err := r.Delete(ctx, ns); ctrlruntimeclient.IgnoreNotFound(err) != nil {
			return err
		}
		if err := kuberneteshelper.TryRemoveFinalizer(ctx, r, externalCluster, kubermaticv1.ExternalClusterKubeOneNamespaceCleanupFinalizer); err != nil {
			log.Errorw("failed to remove kubeone namespace finalizer", zap.Error(err))
			return err
		}
	}
	if kuberneteshelper.HasFinalizer(externalCluster, kubermaticv1.ExternalClusterKubeOneSecretsCleanupFinalizer) {
		if err := r.deleteSecrets(ctx, kubeoneSecrets); err != nil {
			return err
		}

		return kuberneteshelper.TryRemoveFinalizer(ctx, r, externalCluster, kubermaticv1.ExternalClusterKubeOneSecretsCleanupFinalizer)
	}

	return nil
}

func (r *reconciler) updateClusterStatus(ctx context.Context,
	externalCluster *kubermaticv1.ExternalCluster,
	condition kubermaticv1.ExternalClusterCondition) error {
	original := externalCluster.DeepCopy()
	externalCluster.Status.Condition = condition
	kubeoneNamespaceName := externalCluster.GetKubeOneNamespaceName()
	if condition.Phase == kubermaticv1.ExternalClusterPhaseError {
		var phaseError kubermaticv1.ExternalClusterPhase
		podList := &corev1.PodList{}
		err := r.List(ctx,
			podList,
			&ctrlruntimeclient.ListOptions{
				FieldSelector: fields.OneTermEqualSelector(podPhaseKey, string(corev1.PodFailed)),
				Namespace:     kubeoneNamespaceName,
				Limit:         1,
			},
		)
		if err != nil {
			return err
		}
		if len(podList.Items) > 0 {
			failedPod := podList.Items[0]
			statusList := failedPod.Status.ContainerStatuses
			// determine kubeone error using failed pod exitcode.
			if len(statusList) > 0 {
				exitCode := statusList[0].State.Terminated.ExitCode
				phaseError = determineExitCode(exitCode)
				externalCluster.Status.Condition.Phase = phaseError
			}
			// fetch error message from failed pod logs.
			logError, err := getPodLogs(ctx, &failedPod)
			if err != nil {
				return err
			}
			externalCluster.Status.Condition.Message = logError
		}
	}
	if err := r.Patch(ctx, externalCluster, ctrlruntimeclient.MergeFrom(original)); err != nil {
		r.log.Errorw("failed to update external cluster status", zap.Error(err))
		return err
	}
	return nil
}

func (r *reconciler) getKubeOneSecret(ctx context.Context, ref providerconfig.GlobalSecretKeySelector) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: ref.Namespace, Name: ref.Name}, secret); err != nil {
		return nil, err
	}
	return secret, nil
}

func (r *reconciler) createKubeOneNamespace(ctx context.Context, namespace string) error {
	kubeOneNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	if err := r.Create(ctx, kubeOneNamespace); ctrlruntimeclient.IgnoreAlreadyExists(err) != nil {
		return fmt.Errorf("failed to create kubeone cluster namespace: %w", err)
	}

	return nil
}

func getPodLogs(ctx context.Context, pod *corev1.Pod) (string, error) {
	config := ctrlruntime.GetConfigOrDie()
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %w", err)
	}

	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("error in opening stream: %w", err)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf("error in copy information from podLogs to buf: %w", err)
	}
	str := buf.String()

	return str, nil
}

func determineExitCode(exitCode int32) kubermaticv1.ExternalClusterPhase {
	var phaseError kubermaticv1.ExternalClusterPhase
	switch exitCode {
	case fail.RuntimeErrorExitCode:
		phaseError = kubermaticv1.ExternalClusterPhaseRuntimeError
	case fail.EtcdErrorExitCode:
		phaseError = kubermaticv1.ExternalClusterPhaseEtcdError
	case fail.KubeClientErrorExitCode:
		phaseError = kubermaticv1.ExternalClusterPhaseKubeClientError
	case fail.SSHErrorExitCode:
		phaseError = kubermaticv1.ExternalClusterPhaseSSHError
	case fail.ConnectionErrorExitCode:
		phaseError = kubermaticv1.ExternalClusterPhaseConnectionError
	case fail.ConfigErrorExitCode:
		phaseError = kubermaticv1.ExternalClusterPhaseConfigError
	default:
		phaseError = kubermaticv1.ExternalClusterPhaseError
	}
	return phaseError
}
