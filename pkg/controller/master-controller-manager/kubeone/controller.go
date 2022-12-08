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
	"strings"

	"go.uber.org/zap"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubeonev1beta2 "k8c.io/kubeone/pkg/apis/kubeone/v1beta2"
	"k8c.io/kubeone/pkg/fail"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticpred "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sigs.k8s.io/yaml"
)

const (
	// This controller is responsible for managing the lifecycle of KubeOne clusters within KKP.
	ControllerName = "kkp-kubeone-controller"

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

	// KubeOneMigrateJob is the name of kubeone job performing container-runtime migration.
	KubeOneMigrateJob = "kubeone-migrate"

	// KubeOneImportConfigMap is the name of kubeone configmap which stores import action script.
	KubeOneImportConfigMap = "kubeone-import"

	// KubeOneUpgradeConfigMap is the name of kubeone configmap which stores upgrade action script.
	KubeOneUpgradeConfigMap = "kubeone-upgrade"

	// KubeOneMigrateConfigMap is the name of kubeone configmap which stores migrate action script.
	KubeOneMigrateConfigMap = "kubeone-migrate"
)

type reconciler struct {
	ctrlruntimeclient.Client
	log               *zap.SugaredLogger
	secretKeySelector provider.SecretKeySelectorValueFunc
}

func Add(
	ctx context.Context,
	mgr manager.Manager,
	log *zap.SugaredLogger) error {
	reconciler := &reconciler{
		Client:            mgr.GetClient(),
		log:               log.Named(ControllerName),
		secretKeySelector: provider.SecretKeySelectorValueFuncFactory(ctx, mgr.GetClient()),
	}
	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler})
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.ExternalCluster{}},
		&handler.EnqueueRequestForObject{},
		withEventFilter()); err != nil {
		return fmt.Errorf("failed to create externalcluster watcher: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}},
		enqueueExternalCluster(reconciler.Client, reconciler.log),
		updateEventsOnly(),
		ByNameAndNamespace(),
	); err != nil {
		return fmt.Errorf("failed to create kubeone manifest watcher: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &batchv1.Job{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &kubermaticv1.ExternalCluster{},
		},
		updateEventsOnly(),
	); err != nil {
		return fmt.Errorf("failed to create kubeone job watcher: %w", err)
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &corev1.Pod{}, podPhaseKey, func(rawObj ctrlruntimeclient.Object) []string {
		pod := rawObj.(*corev1.Pod)
		return []string{string(pod.Status.Phase)}
	}); err != nil {
		return fmt.Errorf("failed to add index on pod.Status.Phase: %w", err)
	}

	return nil
}

// fetching cluster name using kubeone namespace.
func enqueueExternalCluster(client ctrlruntimeclient.Client, log *zap.SugaredLogger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		var externalClusterName string
		separatedList := strings.Split(a.GetNamespace(), "-")
		if len(separatedList) == 2 {
			externalClusterName = separatedList[1]
		}
		return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: externalClusterName, Namespace: metav1.NamespaceAll}}}
	})
}

func ByNameAndNamespace() predicate.Funcs {
	return kubermaticpred.Factory(func(o ctrlruntimeclient.Object) bool {
		return strings.HasPrefix(o.GetName(), resources.KubeOneManifestSecretPrefix) && strings.HasPrefix(o.GetNamespace(), resources.KubermaticNamespace)
	})
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

func ExternalClusterPausedChecker(ctx context.Context, externalClusterName string, masterClient ctrlruntimeclient.Client) (bool, error) {
	externalCluster := &kubermaticv1.ExternalCluster{}
	if err := masterClient.Get(ctx, types.NamespacedName{Name: externalClusterName}, externalCluster); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get external cluster %q: %w", externalClusterName, err)
	}

	return externalCluster.Spec.Pause, nil
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	paused, err := ExternalClusterPausedChecker(ctx, request.Name, r.Client)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check kubeone external cluster pause status: %w", err)
	}
	if paused {
		return reconcile.Result{}, nil
	}

	log := r.log.With("externalcluster", request.Name)
	log.Info("Processing...")

	return reconcile.Result{}, r.reconcile(ctx, request.Name, log)
}

func (r *reconciler) reconcile(ctx context.Context, externalClusterName string, log *zap.SugaredLogger) error {
	externalCluster := &kubermaticv1.ExternalCluster{}
	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: metav1.NamespaceAll, Name: externalClusterName}, externalCluster); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}

	kubeOneSpec := externalCluster.Spec.CloudSpec.KubeOne
	if kubeOneSpec == nil {
		return nil
	}

	err := r.createKubeOneNamespace(ctx, externalCluster.Name)
	if err != nil {
		return err
	}

	// sync secrets
	projectID := externalCluster.Labels[kubermaticv1.ProjectIDLabelKey]
	if len(projectID) == 0 {
		return fmt.Errorf("externalCluster %s is missing '%s' label", externalCluster.Name, kubermaticv1.ProjectIDLabelKey)
	}
	secrets := &corev1.SecretList{}
	if err := r.List(ctx,
		secrets,
		ctrlruntimeclient.MatchingLabels{kubermaticv1.ProjectIDLabelKey: projectID},
		&ctrlruntimeclient.ListOptions{Namespace: resources.KubermaticNamespace},
	); err != nil {
		return fmt.Errorf("failed to list kubeone secrets in kubermatic namespace: %w", err)
	}

	kubeoneSecrets := []corev1.Secret{}
	for _, secret := range secrets.Items {
		if strings.Contains(secret.Name, externalCluster.Name) {
			kubeoneSecrets = append(kubeoneSecrets, secret)
		}
	}
	if err := r.syncSecrets(ctx, kubeoneSecrets, externalCluster); err != nil {
		return err
	}

	kubeOneNamespace, err := r.getKubeOneNamespace(ctx, externalCluster.Name)
	if err != nil {
		return err
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.Client, externalCluster, kubermaticv1.ExternalClusterKubeOneNamespaceCleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add kubeone namespace finalizer: %w", err)
	}
	if err := kuberneteshelper.TryAddFinalizer(ctx, r.Client, externalCluster, kubermaticv1.ExternalClusterKubeOneCleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add secrets finalizer: %w", err)
	}

	if !externalCluster.DeletionTimestamp.IsZero() {
		log.Info("Deleting KubeOne Namespace and Secrets...")
		if err := r.handleDeletion(ctx, log, kubeOneNamespace, externalCluster, kubeoneSecrets); err != nil {
			return fmt.Errorf("failed deleting kubeone externalcluster: %w", err)
		}
		return nil
	}

	if err := r.importAction(ctx, log, externalCluster); err != nil {
		return err
	}

	if _, err = r.upgradeAction(ctx, log, externalCluster); err != nil {
		return err
	}

	if _, err = r.migrateAction(ctx, log, externalCluster); err != nil {
		return err
	}

	return nil
}

func (r *reconciler) syncSecrets(ctx context.Context, secrets []corev1.Secret, externalCluster *kubermaticv1.ExternalCluster) error {
	for _, secret := range secrets {
		if _, err := r.CreateOrUpdateSecretForCluster(ctx, externalCluster, secret.Data, secret.Name, GetKubeOneNamespaceName(externalCluster.Name)); ctrlruntimeclient.IgnoreAlreadyExists(err) != nil {
			return err
		}
	}
	return nil
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
	externalCluster *kubermaticv1.ExternalCluster) error {
	if externalCluster.Spec.KubeconfigReference == nil {
		_, err := r.initiateImportCluster(ctx, log, externalCluster)
		if err != nil {
			log.Errorw("failed to import kubeone cluster", zap.Error(err))
			return err
		}
	}
	return nil
}

func (r *reconciler) initiateImportCluster(ctx context.Context,
	log *zap.SugaredLogger,
	externalCluster *kubermaticv1.ExternalCluster) (*kubermaticv1.ExternalCluster, error) {
	log.Info("Importing kubeone luster...")

	kubeoneNamespace := kubernetesprovider.GetKubeOneNamespaceName(externalCluster.Name)

	if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterPhaseProvisioning); err != nil {
		return nil, err
	}

	log.Info("Generating kubeone job to fetch kubeconfig...")
	job, err := r.generateKubeOneActionJob(ctx, log, externalCluster, ImportAction)
	if err != nil {
		return nil, fmt.Errorf("could not generate kubeone job: %w", err)
	}

	log.Info("Creating kubeone job to fetch kubeconfig...")
	if err := r.Create(ctx, job); ctrlruntimeclient.IgnoreAlreadyExists(err) != nil {
		return nil, fmt.Errorf("could not create kubeone job %s/%s: %w", job.Name, job.Namespace, err)
	}

	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: job.Namespace, Name: job.Name}, job); err != nil {
		return nil, fmt.Errorf("failed to get kubeone kubeconfig job: %w", err)
	}

	if job.Status.Active > 0 {
		log.Info("kubeone import job active")
		return nil, nil
	}

	// job failed.
	if job.Status.Succeeded == 0 && job.Status.Failed >= 1 {
		// update kubeone externalcluster status.
		log.Info("kubeone import failed!")
		if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterPhaseError); err != nil {
			return nil, err
		}
		// delete kubeone job alongwith its pods as no longer required.
		propagationPolicy := metav1.DeletePropagationBackground
		err = r.Delete(ctx, job, &ctrlruntimeclient.DeleteOptions{PropagationPolicy: &propagationPolicy})
		if err != nil {
			return nil, err
		}
		return nil, errors.New("kubeone import failed")
	}

	// job succeeded.
	podList := &corev1.PodList{}
	err = r.List(ctx,
		podList,
		&ctrlruntimeclient.ListOptions{
			Limit:         1,
			Namespace:     job.Namespace,
			FieldSelector: fields.OneTermEqualSelector(podPhaseKey, string(corev1.PodSucceeded)),
		},
		&ctrlruntimeclient.MatchingLabels{"job-name": job.Name},
	)
	if err != nil {
		return nil, err
	}

	if len(podList.Items) == 0 {
		return nil, fmt.Errorf("no succeeded import pods in kubeone namespace:%s", kubeoneNamespace)
	}

	succeededPod := podList.Items[0]
	// use pod logs to get cluster's kubeconfig.
	config, err := getPodLogs(ctx, &succeededPod)
	if err != nil {
		return nil, err
	}

	log.Info("fetched config from the kubeone pod")

	secretName := externalCluster.GetKubeconfigSecretName()
	secretNamespace := resources.KubermaticNamespace

	data := map[string][]byte{
		resources.ExternalClusterKubeconfig: []byte(config),
	}
	kubeconfigRef, err := r.CreateOrUpdateSecretForCluster(ctx, externalCluster, data, secretName, secretNamespace)
	if err != nil {
		return nil, err
	}

	oldexternalCluster := externalCluster.DeepCopy()
	externalCluster.Spec.KubeconfigReference = kubeconfigRef
	if err := r.Patch(ctx, externalCluster, ctrlruntimeclient.MergeFrom(oldexternalCluster)); err != nil {
		log.Errorw("failed to add kubeconfig reference in external cluster", zap.Error(err))
		return nil, err
	}

	log.Info("KubeOne Cluster Imported!")
	if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterPhaseRunning); err != nil {
		return nil, err
	}
	// delete kubeone job alongwith its pods as no longer required.
	propagationPolicy := metav1.DeletePropagationBackground
	err = r.Delete(ctx, job, &ctrlruntimeclient.DeleteOptions{PropagationPolicy: &propagationPolicy})
	if err != nil {
		return nil, err
	}

	return externalCluster, nil
}

func (r *reconciler) CreateOrUpdateSecretForCluster(ctx context.Context, externalcluster *kubermaticv1.ExternalCluster, secretData map[string][]byte, secretName, secretNamespace string) (*providerconfig.GlobalSecretKeySelector, error) {
	reconciler, err := credentialSecretReconcilerFactory(secretName, externalcluster.Labels, secretData)
	if err != nil {
		return nil, err
	}

	if err := reconciling.ReconcileSecrets(ctx, []reconciling.NamedSecretReconcilerFactory{reconciler}, secretNamespace, r); err != nil {
		return nil, fmt.Errorf("failed to ensure Secret: %w", err)
	}

	return &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      secretName,
			Namespace: secretNamespace,
		},
	}, nil
}

func credentialSecretReconcilerFactory(secretName string, clusterLabels map[string]string, secretData map[string][]byte) (reconciling.NamedSecretReconcilerFactory, error) {
	projectID := clusterLabels[kubermaticv1.ProjectIDLabelKey]
	if len(projectID) == 0 {
		return nil, fmt.Errorf("cluster is missing '%s' label", kubermaticv1.ProjectIDLabelKey)
	}

	return func() (name string, reconciler reconciling.SecretReconciler) {
		return secretName, func(existing *corev1.Secret) (*corev1.Secret, error) {
			if existing.Labels == nil {
				existing.Labels = map[string]string{}
			}

			existing.Labels[kubermaticv1.ProjectIDLabelKey] = projectID
			existing.Data = secretData

			return existing, nil
		}
	}, nil
}

func (r *reconciler) upgradeAction(ctx context.Context,
	log *zap.SugaredLogger,
	externalCluster *kubermaticv1.ExternalCluster) (*batchv1.Job, error) {
	manifestRef := externalCluster.Spec.CloudSpec.KubeOne.ManifestReference
	kubeOneNamespaceName := kubernetesprovider.GetKubeOneNamespaceName(externalCluster.Name)

	manifestSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: kubeOneNamespaceName, Name: manifestRef.Name}, manifestSecret); err != nil {
		return nil, nil
	}
	currentManifest := manifestSecret.Data[resources.KubeOneManifest]

	clusterClient, err := kuberneteshelper.GetClusterClient(ctx, externalCluster, r.Client)
	if err != nil {
		return nil, err
	}

	version, err := kuberneteshelper.GetVersion(clusterClient)
	if err != nil {
		return nil, err
	}
	currentVersion := version.String()
	desiredVersion, err := getDesiredVersion(currentManifest)
	if err != nil {
		return nil, err
	}

	isdesiredPhase := sets.NewString(string(kubermaticv1.ExternalClusterPhaseError), string(kubermaticv1.ExternalClusterPhaseRunning)).Has(string(externalCluster.Status.Condition.Phase))
	// proceed only when desiredVersion is different from currentVersion or clusterphase is desiredPhase.
	if currentVersion == desiredVersion || !isdesiredPhase {
		return nil, nil
	}

	log.Infow("Upgrading kubeone cluster...", "from", currentVersion, "to", desiredVersion)
	if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterPhaseReconciling); err != nil {
		return nil, err
	}

	upgradeJob, err := r.initiateClusterUpgrade(ctx, log, currentVersion, desiredVersion, externalCluster)
	if err != nil {
		log.Errorw("failed to upgrade kubeone cluster", zap.Error(err))
		return nil, err
	}

	return upgradeJob, nil
}

func (r *reconciler) initiateClusterUpgrade(ctx context.Context,
	log *zap.SugaredLogger,
	currentVersion, desiredVersion string,
	cluster *kubermaticv1.ExternalCluster) (*batchv1.Job, error) {
	log.Info("Upgrading kubeone cluster...")

	if err := r.updateClusterStatus(ctx, cluster, kubermaticv1.ExternalClusterPhaseReconciling); err != nil {
		return nil, err
	}

	job, err := r.generateKubeOneActionJob(ctx, log, cluster, UpgradeControlPlaneAction)
	if err != nil {
		return nil, err
	}

	log.Info("Creating kubeone job to upgrade kubeone...")
	if err := r.Create(ctx, job); ctrlruntimeclient.IgnoreAlreadyExists(err) != nil {
		return nil, err
	}

	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: job.Namespace, Name: job.Name}, job); err != nil {
		return nil, fmt.Errorf("failed to get kubeone upgrade job: %w", err)
	}

	if job.Status.Active > 0 {
		log.Info("kubeone upgrade job active")
		return nil, nil
	}

	// job failed.
	if job.Status.Succeeded == 0 && job.Status.Failed >= 1 {
		log.Info("kubeone upgrade failed...")
		if err := r.updateClusterStatus(ctx, cluster, kubermaticv1.ExternalClusterPhaseError); err != nil {
			return nil, err
		}
		// delete kubeone job alongwith its pods as no longer required.
		propagationPolicy := metav1.DeletePropagationBackground
		err = r.Delete(ctx, job, &ctrlruntimeclient.DeleteOptions{PropagationPolicy: &propagationPolicy})
		if err != nil {
			return nil, err
		}
		return nil, errors.New("kubeone upgrade failed")
	}

	if currentVersion != desiredVersion {
		return nil, errors.New("kubeone upgrade job succeeded but desiredVersion != currentVersion")
	}
	log.Info("KubeOne Cluster Upgraded!")
	if err := r.updateClusterStatus(ctx, cluster, kubermaticv1.ExternalClusterPhaseRunning); err != nil {
		return nil, err
	}
	// delete kubeone job alongwith its pods as no longer required.
	propagationPolicy := metav1.DeletePropagationBackground
	err = r.Delete(ctx, job, &ctrlruntimeclient.DeleteOptions{PropagationPolicy: &propagationPolicy})
	if err != nil {
		return nil, err
	}

	return job, nil
}

func (r *reconciler) migrateAction(ctx context.Context,
	log *zap.SugaredLogger,
	externalCluster *kubermaticv1.ExternalCluster) (*batchv1.Job, error) {
	manifestRef := externalCluster.Spec.CloudSpec.KubeOne.ManifestReference
	kubeOneNamespace := manifestRef.Namespace
	manifestSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: kubeOneNamespace, Name: manifestRef.Name}, manifestSecret); err != nil {
		log.Errorw("can not retrieve kubeone manifest secret", zap.Error(err))
		return nil, err
	}
	currentManifest := manifestSecret.Data[resources.KubeOneManifest]

	clusterClient, err := kuberneteshelper.GetClusterClient(ctx, externalCluster, r.Client)
	if err != nil {
		return nil, err
	}
	currentContainerRuntime, err := kuberneteshelper.CheckContainerRuntime(ctx, clusterClient)
	if err != nil {
		return nil, err
	}
	desiredContainerRuntime, err := getDesiredContainerRuntime(currentManifest)
	if err != nil {
		return nil, err
	}

	isdesiredPhase := sets.NewString(string(kubermaticv1.ExternalClusterPhaseError), string(kubermaticv1.ExternalClusterPhaseRunning)).Has(string(externalCluster.Status.Condition.Phase))
	if !isdesiredPhase || currentContainerRuntime == desiredContainerRuntime || desiredContainerRuntime != resources.ContainerRuntimeContainerd {
		return nil, nil
	}

	log.Infow("Migrating kubeone cluster container runtime...", "from", currentContainerRuntime, "to", desiredContainerRuntime)
	migratePod, err := r.initiateClusterMigration(ctx, log, currentContainerRuntime, desiredContainerRuntime, externalCluster)
	if err != nil {
		log.Errorw("failed to migrate kubeone cluster", zap.Error(err))
		return nil, err
	}

	return migratePod, nil
}

func (r *reconciler) initiateClusterMigration(ctx context.Context,
	log *zap.SugaredLogger,
	currentContainerRuntime, desiredContainerRuntime string,
	cluster *kubermaticv1.ExternalCluster) (*batchv1.Job, error) {
	log.Info("Migrating kubeone cluster...")

	if err := r.updateClusterStatus(ctx, cluster, kubermaticv1.ExternalClusterPhaseReconciling); err != nil {
		return nil, err
	}

	job, err := r.generateKubeOneActionJob(ctx, log, cluster, MigrateContainerRuntimeAction)
	if err != nil {
		return nil, fmt.Errorf("could not generate kubeone pod %s/%s to migrate container runtime: %w", job.Name, job.Namespace, err)
	}

	log.Info("Creating kubeone job to migrate kubeone...")
	if err := r.Create(ctx, job); ctrlruntimeclient.IgnoreAlreadyExists(err) != nil {
		return nil, fmt.Errorf("could not create kubeone job %s/%s to migrate kubeone cluster: %w", job.Name, job.Namespace, err)
	}

	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: job.Namespace, Name: job.Name}, job); err != nil {
		return nil, fmt.Errorf("failed to get kubeone upgrade job: %w", err)
	}

	if job.Status.Active > 0 {
		log.Info("kubeone migrate job active")
		return nil, nil
	}

	// job failed.
	if job.Status.Succeeded == 0 && job.Status.Failed >= 1 {
		if err := r.updateClusterStatus(ctx, cluster, kubermaticv1.ExternalClusterPhaseError); err != nil {
			return nil, err
		}
		// delete kubeone job alongwith its pods as no longer required.
		propagationPolicy := metav1.DeletePropagationBackground
		err = r.Delete(ctx, job, &ctrlruntimeclient.DeleteOptions{PropagationPolicy: &propagationPolicy})
		if err != nil {
			return nil, err
		}
		return nil, errors.New("kubeone migration failed")
	}

	if currentContainerRuntime != desiredContainerRuntime {
		return nil, fmt.Errorf("kubeone migrate job succeeded but currentContainerRuntime != desiredContainerRuntime")
	}
	log.Info("KubeOne Cluster Migrated!")
	if err := r.updateClusterStatus(ctx, cluster, kubermaticv1.ExternalClusterPhaseRunning); err != nil {
		return nil, err
	}
	// delete kubeone job alongwith its pods as no longer required.
	propagationPolicy := metav1.DeletePropagationBackground
	err = r.Delete(ctx, job, &ctrlruntimeclient.DeleteOptions{PropagationPolicy: &propagationPolicy})
	if err != nil {
		return nil, err
	}

	return job, nil
}

func (r *reconciler) generateKubeOneActionJob(ctx context.Context, log *zap.SugaredLogger, externalCluster *kubermaticv1.ExternalCluster, action string) (*batchv1.Job, error) {
	var kubeoneJobName, kubeoneCMName string
	var sshSecret, manifestSecret *corev1.Secret
	var err error
	kubeOne := externalCluster.Spec.CloudSpec.KubeOne
	envVar := []corev1.EnvVar{}
	volumes := []corev1.Volume{}
	providerName := kubeOne.ProviderName
	kubeOneNamespaceName := kubernetesprovider.GetKubeOneNamespaceName(externalCluster.Name)

	if kubeOne.SSHReference != nil {
		sshSecret, err = r.getKubeOneSecret(ctx, *kubeOne.SSHReference)
		if err != nil {
			log.Errorw("could not find kubeone ssh secret", zap.Error(err))
			return nil, err
		}
	}
	if kubeOne.ManifestReference != nil {
		manifestSecret, err = r.getKubeOneSecret(ctx, *kubeOne.ManifestReference)
		if err != nil {
			log.Errorw("could not find kubeone manifest secret", zap.Error(err))
			return nil, err
		}
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

	switch {
	case action == ImportAction:
		kubeoneJobName = KubeOneImportJob
		kubeoneCMName = KubeOneImportConfigMap
	case action == UpgradeControlPlaneAction:
		kubeoneJobName = KubeOneUpgradeJob
		kubeoneCMName = KubeOneUpgradeConfigMap
	case action == MigrateContainerRuntimeAction:
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
					Controller: pointer.Bool(true),
					UID:        externalCluster.GetUID(),
				},
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							Name:       externalCluster.Name,
							APIVersion: kubermaticv1.SchemeGroupVersion.String(),
							Kind:       kubermaticv1.ExternalClusterKind,
							Controller: pointer.Bool(true),
							UID:        externalCluster.GetUID(),
						},
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:    "copy-ro-manifest",
							Image:   "busybox",
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
							Image:   fmt.Sprintf("%s:%s", resources.KubeOneImage, resources.KubeOneImageTag),
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
									DefaultMode: pointer.Int32(256),
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
									DefaultMode: pointer.Int32(448),
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
	if providerName == resources.KubeOneEquinix {
		envVarSource.SecretKeyRef.Key = resources.PacketAPIKey
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "PACKET_API_KEY",
				ValueFrom: envVarSource,
			},
		)
		envVarSource.SecretKeyRef.Key = resources.PacketProjectID
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:      "PACKET_PROJECT_ID",
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

	switch {
	case action == ImportAction:
		name = KubeOneImportConfigMap
		scriptToRun += "kubeone kubeconfig --manifest kubeonemanifest/manifest"
	case action == UpgradeControlPlaneAction:
		name = KubeOneUpgradeConfigMap
		scriptToRun += "kubeone apply --manifest kubeonemanifest/manifest -y --log-format json"
	case action == MigrateContainerRuntimeAction:
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
	if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterPhaseDeleting); err != nil {
		return err
	}
	if kuberneteshelper.HasFinalizer(externalCluster, kubermaticv1.ExternalClusterKubeOneNamespaceCleanupFinalizer) {
		if err := r.Delete(ctx, ns); err != nil {
			return err
		}
		if err := kuberneteshelper.TryRemoveFinalizer(ctx, r, externalCluster, kubermaticv1.ExternalClusterKubeOneNamespaceCleanupFinalizer); err != nil {
			log.Errorw("failed to remove kubeone namespace finalizer", zap.Error(err))
			return err
		}
	}
	if kuberneteshelper.HasFinalizer(externalCluster, kubermaticv1.ExternalClusterKubeOneCleanupFinalizer) {
		if err := r.deleteSecrets(ctx, kubeoneSecrets); err != nil {
			return err
		}

		return kuberneteshelper.TryRemoveFinalizer(ctx, r, externalCluster, kubermaticv1.ExternalClusterKubeOneCleanupFinalizer)
	}

	return nil
}

func (r *reconciler) updateClusterStatus(ctx context.Context,
	externalCluster *kubermaticv1.ExternalCluster,
	phase kubermaticv1.ExternalClusterPhase) error {
	oldexternalCluster := externalCluster.DeepCopy()
	externalCluster.Status.Condition.Phase = phase
	if phase == kubermaticv1.ExternalClusterPhaseError {
		var phaseError kubermaticv1.ExternalClusterPhase
		// fetch failed pod, assuming only one failed pod in namespace as deleting jobs after each operation.
		podList := &corev1.PodList{}
		err := r.List(ctx,
			podList,
			&ctrlruntimeclient.ListOptions{
				FieldSelector: fields.OneTermEqualSelector(podPhaseKey, string(corev1.PodFailed)),
				Namespace:     kubernetesprovider.GetKubeOneNamespaceName(externalCluster.Name),
			},
		)
		if err != nil {
			return err
		}
		if len(podList.Items) == 0 {
			return fmt.Errorf("no failed pods in kubeone namespace: %s", kubernetesprovider.GetKubeOneNamespaceName(externalCluster.Name))
		}
		failedPod := podList.Items[0]
		statusList := failedPod.Status.ContainerStatuses
		// determine kubeone error using failed pod exitcode.
		if len(statusList) > 0 {
			exitCode := statusList[0].State.Terminated.ExitCode
			phaseError = determineExitCode(exitCode)
		}
		// fetch error message from failed pod logs.
		logError, err := getPodLogs(ctx, &failedPod)
		if err != nil {
			return err
		}
		externalCluster.Status.Condition = kubermaticv1.ExternalClusterCondition{
			Phase:   phaseError,
			Message: logError,
		}
	}
	if err := r.Patch(ctx, externalCluster, ctrlruntimeclient.MergeFrom(oldexternalCluster)); err != nil {
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

func GetKubeOneNamespaceName(externalClusterName string) string {
	return fmt.Sprintf("%s-%s", resources.KubeOneNamespacePrefix, externalClusterName)
}

func (r *reconciler) createKubeOneNamespace(ctx context.Context, extexternalClusterName string) error {
	kubeOneNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: kubernetesprovider.GetKubeOneNamespaceName(extexternalClusterName),
		},
	}
	if err := r.Create(ctx, kubeOneNamespace); ctrlruntimeclient.IgnoreAlreadyExists(err) != nil {
		return fmt.Errorf("failed to create kubeone cluster namespace: %w", err)
	}

	return nil
}

func (r *reconciler) getKubeOneNamespace(ctx context.Context, extexternalClusterName string) (*corev1.Namespace, error) {
	ns := &corev1.Namespace{}
	name := types.NamespacedName{Name: kubernetesprovider.GetKubeOneNamespaceName(extexternalClusterName)}
	if err := r.Get(ctx, name, ns); err != nil {
		return nil, err
	}
	return ns, nil
}

func getDesiredVersion(currentManifest []byte) (string, error) {
	cluster := &kubeonev1beta2.KubeOneCluster{}
	if err := yaml.UnmarshalStrict(currentManifest, cluster); err != nil {
		return "", fmt.Errorf("failed to decode manifest secret data: %w", err)
	}

	return cluster.Versions.Kubernetes, nil
}

func getDesiredContainerRuntime(currentManifest []byte) (string, error) {
	cluster := &kubeonev1beta2.KubeOneCluster{}
	if err := yaml.UnmarshalStrict(currentManifest, cluster); err != nil {
		return "", fmt.Errorf("failed to decode manifest secret data: %w", err)
	}
	if cluster.ContainerRuntime.Docker != nil {
		return resources.ContainerRuntimeDocker, nil
	} else if cluster.ContainerRuntime.Containerd != nil {
		return resources.ContainerRuntimeContainerd, nil
	}
	return "", nil
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
	switch {
	case exitCode == fail.RuntimeErrorExitCode:
		phaseError = kubermaticv1.ExternalClusterPhaseRuntimeError
	case exitCode == fail.EtcdErrorExitCode:
		phaseError = kubermaticv1.ExternalClusterPhaseEtcdError
	case exitCode == fail.KubeClientErrorExitCode:
		phaseError = kubermaticv1.ExternalClusterPhaseKubeClientError
	case exitCode == fail.SSHErrorExitCode:
		phaseError = kubermaticv1.ExternalClusterPhaseSSHError
	case exitCode == fail.ConnectionErrorExitCode:
		phaseError = kubermaticv1.ExternalClusterPhaseConnectionError
	case exitCode == fail.ConfigErrorExitCode:
		phaseError = kubermaticv1.ExternalClusterPhaseConfigError
	default:
		phaseError = kubermaticv1.ExternalClusterPhaseError
	}
	return phaseError
}
