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
	"encoding/json"
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
	"k8c.io/kubermatic/v2/pkg/util/restmapper"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
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

	// ImportAction is the action to import kubeone cluster.
	ImportAction = "import"

	// UpgradeControlPlaneAction is the action to upgrade kubeone cluster.
	UpgradeControlPlaneAction = "upgrade"

	// MigrateContainerRuntimeAction is the action to migrate kubeone container-runtime.
	MigrateContainerRuntimeAction = "migrate"

	// KubeOneImportPod is the name of kubeone pod performing import.
	KubeOneImportPod = "kubeone-import"

	// KubeOneUpgradePod is the name of kubeone pod performing upgrade.
	KubeOneUpgradePod = "kubeone-upgrade"

	// KubeOneMigratePod is the name of kubeone pod performing container-runtime migration.
	KubeOneMigratePod = "kubeone-migrate"

	// KubeOneImportConfigMap is the name of kubeone configmap which stores import action script.
	KubeOneImportConfigMap = "kubeone-import"

	// KubeOneUpgradeConfigMap is the name of kubeone configmap which stores upgrade action script.
	KubeOneUpgradeConfigMap = "kubeone-upgrade"

	// KubeOneMigrateConfigMap is the name of kubeone configmap which stores migrate action script.
	KubeOneMigrateConfigMap = "kubeone-migrate"

	// KubeOneLogLevelError is kubeone log level for errors.
	KubeOneLogLevelError = "error"
)

type reconciler struct {
	ctrlruntimeclient.Client
	log *zap.SugaredLogger
	kubernetesprovider.ImpersonationClient
	secretKeySelector provider.SecretKeySelectorValueFunc
}

type kubeOneLog struct {
	Level   string `json:"level"`
	Message string `json:"msg"`
}

func Add(
	ctx context.Context,
	mgr manager.Manager,
	log *zap.SugaredLogger) error {
	reconciler := &reconciler{
		Client:              mgr.GetClient(),
		log:                 log.Named(ControllerName),
		ImpersonationClient: kubernetesprovider.NewImpersonationClient(mgr.GetConfig(), mgr.GetRESTMapper()).CreateImpersonatedClient,
		secretKeySelector:   provider.SecretKeySelectorValueFuncFactory(ctx, mgr.GetClient()),
	}
	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler})
	if err != nil {
		return err
	}
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.ExternalCluster{}}, &handler.EnqueueRequestForObject{}, withEventFilter()); err != nil {
		return fmt.Errorf("failed to create externalcluster watcher: %w", err)
	}
	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}},
		enqueueExternalCluster(reconciler.Client, reconciler.log),
		updateEventsOnly(),
		ByNameAndNamespace(),
	); err != nil {
		return fmt.Errorf("failed to create kubeone manifest watcher: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &corev1.Pod{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &kubermaticv1.ExternalCluster{},
		},
		updateEventsOnly(),
	); err != nil {
		return fmt.Errorf("failed to create kubeone pod watcher: %w", err)
	}

	return nil
}

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
		return o.GetName() == resources.KubeOneManifestSecretName && strings.HasPrefix(o.GetNamespace(), resources.KubeOneNamespacePrefix)
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
			if externalCluster.Spec.CloudSpec == nil {
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
	log := r.log.With("cluster", request.Name)
	log.Info("Processing...")

	externalCluster := &kubermaticv1.ExternalCluster{}
	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: metav1.NamespaceAll, Name: request.Name}, externalCluster); err != nil {
		if !apierrors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
	}

	if externalCluster.DeletionTimestamp != nil {
		log.Info("Deleting KubeOne Namespace...")
		ns := &corev1.Namespace{}
		name := types.NamespacedName{Name: kubernetesprovider.GetKubeOneNamespaceName(externalCluster.Name)}
		if err := r.Get(ctx, name, ns); err != nil {
			if !apierrors.IsNotFound(err) {
				return reconcile.Result{}, err
			}
		}
		if err := r.Delete(ctx, ns); err != nil {
			return reconcile.Result{}, err
		}

		if err := kuberneteshelper.TryRemoveFinalizer(ctx, r, externalCluster, kubermaticv1.ExternalClusterKubeOneNamespaceCleanupFinalizer); err != nil {
			log.Errorw("failed to remove kubeone namespace finalizer", zap.Error(err))
			return reconcile.Result{}, err
		}
	}
	return r.reconcile(ctx, log, externalCluster)
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, externalCluster *kubermaticv1.ExternalCluster) (reconcile.Result, error) {
	cloud := externalCluster.Spec.CloudSpec
	if cloud == nil || cloud.KubeOne == nil {
		return reconcile.Result{}, nil
	}
	clusterPhase := externalCluster.Status.Condition.Phase

	// return if cluster in Error phase
	if strings.HasSuffix(string(clusterPhase), "Error") {
		return reconcile.Result{}, nil
	}

	if err := r.initiateImportAction(ctx, log, externalCluster); err != nil {
		return reconcile.Result{}, err
	}

	externalClusterProvider, err := kubernetesprovider.NewExternalClusterProvider(r.ImpersonationClient, r.Client)
	if err != nil {
		return reconcile.Result{}, nil
	}

	manifestRef := cloud.KubeOne.ManifestReference
	kubeOneNamespace := manifestRef.Namespace
	manifestSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: kubeOneNamespace, Name: manifestRef.Name}, manifestSecret); err != nil {
		log.Errorw("can not retrieve kubeone manifest secret", zap.Error(err))
		return reconcile.Result{}, nil
	}
	currentManifest := manifestSecret.Data[resources.KubeOneManifest]

	if _, err = r.initiateUpgradeAction(ctx, log, externalCluster, externalClusterProvider, currentManifest, clusterPhase); err != nil {
		return reconcile.Result{}, err
	}

	if err = r.checkPodStatusIfExists(ctx, log, externalCluster, UpgradeControlPlaneAction, KubeOneUpgradePod); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to upgrade kubeone cluster: %w", err)
	}

	if _, err = r.initiateMigrateAction(ctx, log, externalCluster, externalClusterProvider, currentManifest, clusterPhase); err != nil {
		return reconcile.Result{}, err
	}

	if err = r.checkPodStatusIfExists(ctx, log, externalCluster, MigrateContainerRuntimeAction, KubeOneMigratePod); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to migrate kubeone cluster: %w", err)
	}

	return reconcile.Result{}, nil
}

func (r *reconciler) initiateImportAction(
	ctx context.Context,
	log *zap.SugaredLogger,
	externalCluster *kubermaticv1.ExternalCluster) error {
	if externalCluster.Spec.KubeconfigReference == nil {
		externalCluster, err := r.importCluster(ctx, log, externalCluster)
		if err != nil {
			log.Errorw("failed to import kubeone cluster", zap.Error(err))
			return err
		}
		// update kubeone externalcluster status.
		if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterCondition{
			Phase: kubermaticv1.ExternalClusterPhaseRunning,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *reconciler) importCluster(ctx context.Context, log *zap.SugaredLogger, externalCluster *kubermaticv1.ExternalCluster) (*kubermaticv1.ExternalCluster, error) {
	log.Info("Importing kubeone cluster...")

	log.Info("Generating kubeone pod to fetch kubeconfig...")
	generatedPod, err := r.generateKubeOneActionPod(ctx, log, externalCluster, ImportAction)
	if err != nil {
		return nil, fmt.Errorf("could not generate kubeone pod: %w", err)
	}

	log.Info("Creating kubeone pod to fetch kubeconfig...")
	if err := r.Create(ctx, generatedPod); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("could not create kubeone pod %s/%s: %w", KubeOneImportPod, generatedPod.Namespace, err)
		}
	}

	// fetch kubeone pod status till its completion
	for generatedPod.Status.Phase != corev1.PodSucceeded {
		if generatedPod.Status.Phase == corev1.PodFailed {
			var phaseError kubermaticv1.ExternalClusterPhase
			// fetch kubeone error code
			statusList := generatedPod.Status.ContainerStatuses
			if len(statusList) > 0 {
				exitCode := statusList[0].State.Terminated.ExitCode
				phaseError = determineExitCode(exitCode)
			}

			logError, err := getPodLogs(ctx, generatedPod)
			if err != nil {
				return nil, err
			}
			if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterCondition{
				Phase:   phaseError,
				Message: logError,
			}); err != nil {
				return nil, err
			}

			return nil, errors.New(logError)
		}
		if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: generatedPod.Namespace, Name: KubeOneImportPod}, generatedPod); err != nil {
			return nil, fmt.Errorf("failed to get kubeone kubeconfig pod: %w", err)
		}
	}

	config, err := getPodLogs(ctx, generatedPod)
	if err != nil {
		return nil, err
	}

	if err := verifyKubeconfig(ctx, config); err != nil {
		return nil, err
	}

	kubeconfigRef, err := r.CreateOrUpdateKubeconfigSecretForCluster(ctx, log, externalCluster, config, generatedPod.Namespace)
	if err != nil {
		return nil, err
	}
	oldexternalCluster := externalCluster.DeepCopy()
	externalCluster.Spec.KubeconfigReference = kubeconfigRef
	if err := r.Patch(ctx, externalCluster, ctrlruntimeclient.MergeFrom(oldexternalCluster)); err != nil {
		log.Errorw("failed to add kubeconfig reference in external cluster", zap.Error(err))
		return nil, err
	}

	// cleanup pod as no longer required.
	err = r.Delete(ctx, generatedPod)
	if err != nil {
		return nil, err
	}

	log.Info("KubeOne Cluster Imported!")
	return externalCluster, nil
}

func verifyKubeconfig(ctx context.Context, config string) error {
	// generate kubeconfig for cluster.
	cfg, err := clientcmd.Load([]byte(config))
	if err != nil {
		return err
	}
	clientset, err := GenerateClient(cfg)
	if err != nil {
		return err
	}
	// connect to the kubernetes cluster.
	err = clientset.List(ctx, &corev1.NodeList{})
	if err != nil {
		return err
	}
	return nil
}

func (r *reconciler) initiateUpgradeAction(ctx context.Context,
	log *zap.SugaredLogger,
	externalCluster *kubermaticv1.ExternalCluster,
	externalClusterProvider *kubernetesprovider.ExternalClusterProvider,
	currentManifest []byte,
	clusterPhase kubermaticv1.ExternalClusterPhase,
) (*corev1.Pod, error) {
	version, err := externalClusterProvider.GetVersion(ctx, externalCluster)
	if err != nil {
		return nil, err
	}
	currentVersion := version.String()
	desiredVersion, err := getDesiredVersion(currentManifest)
	if err != nil {
		return nil, err
	}
	if clusterPhase != kubermaticv1.ExternalClusterPhaseRunning || currentVersion == desiredVersion {
		return nil, nil
	}
	log.Infow("Upgrading kubeone cluster...", "from", currentVersion, "to", desiredVersion)

	upgradePod, err := r.upgradeCluster(ctx, log, externalCluster)
	if err != nil {
		log.Errorw("failed to upgrade kubeone cluster", zap.Error(err))
		return nil, err
	}
	// update kubeone externalcluster status.
	if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterCondition{
		Phase: kubermaticv1.ExternalClusterPhaseReconciling,
	}); err != nil {
		return nil, err
	}

	return upgradePod, nil
}

func (r *reconciler) initiateMigrateAction(ctx context.Context,
	log *zap.SugaredLogger,
	externalCluster *kubermaticv1.ExternalCluster,
	externalClusterProvider *kubernetesprovider.ExternalClusterProvider,
	currentManifest []byte,
	clusterPhase kubermaticv1.ExternalClusterPhase,
) (*corev1.Pod, error) {
	currentContainerRuntime, err := kuberneteshelper.CheckContainerRuntime(ctx, externalCluster, externalClusterProvider)
	if err != nil {
		return nil, err
	}
	desiredContainerRuntime, err := getDesiredContainerRuntime(currentManifest)
	if err != nil {
		return nil, err
	}

	if clusterPhase != kubermaticv1.ExternalClusterPhaseRunning || currentContainerRuntime == desiredContainerRuntime || desiredContainerRuntime != resources.ContainerRuntimeContainerd {
		return nil, nil
	}

	log.Infow("Migrating kubeone cluster container runtime...", "from", currentContainerRuntime, "to", desiredContainerRuntime)
	migratePod, err := r.migrateCluster(ctx, log, externalCluster)
	if err != nil {
		log.Errorw("failed to migrate kubeone cluster", zap.Error(err))
		return nil, err
	}
	// update kubeone externalcluster status.
	if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterCondition{
		Phase: kubermaticv1.ExternalClusterPhaseReconciling,
	}); err != nil {
		return nil, err
	}
	return migratePod, nil
}

func getDesiredVersion(currentManifest []byte) (string, error) {
	cluster := &kubeonev1beta2.KubeOneCluster{}
	if err := yaml.UnmarshalStrict(currentManifest, cluster); err != nil {
		return "", fmt.Errorf("failed to decode manifest secret data: %w", err)
	}

	return cluster.Versions.Kubernetes, nil
}

func (r *reconciler) checkPodStatusIfExists(ctx context.Context,
	log *zap.SugaredLogger,
	externalCluster *kubermaticv1.ExternalCluster,
	action,
	podName string) error {
	pod := &corev1.Pod{}

	err := r.Get(ctx, ctrlruntimeclient.ObjectKey{
		Name:      podName,
		Namespace: kubernetesprovider.GetKubeOneNamespaceName(externalCluster.Name),
	},
		pod,
	)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	} else {
		err = r.checkPodStatus(ctx, log, pod, externalCluster, action)
		if err != nil {
			return err
		}
	}
	return nil
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

func (r *reconciler) updateClusterStatus(ctx context.Context,
	externalCluster *kubermaticv1.ExternalCluster,
	status kubermaticv1.ExternalClusterCondition) error {
	oldexternalCluster := externalCluster.DeepCopy()
	externalCluster.Status.Condition = status
	if err := r.Patch(ctx, externalCluster, ctrlruntimeclient.MergeFrom(oldexternalCluster)); err != nil {
		r.log.Errorw("failed to update external cluster status", zap.Error(err))
		return err
	}
	return nil
}

func (r *reconciler) checkPodStatus(ctx context.Context,
	log *zap.SugaredLogger,
	pod *corev1.Pod,
	externalCluster *kubermaticv1.ExternalCluster,
	action string,
) error {
	log.Infow("Checking kubeone pod status...", "Pod", pod.Name)
	if pod.Status.Phase == corev1.PodSucceeded {
		// update kubeone externalcluster status.
		if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterCondition{
			Phase: kubermaticv1.ExternalClusterPhaseRunning,
		}); err != nil {
			return err
		}
		if action == UpgradeControlPlaneAction {
			log.Info("KubeOne Cluster Upgraded!")
		} else if action == MigrateContainerRuntimeAction {
			log.Info("KubeOne Cluster Migrated!")
		}
		if err := r.Delete(ctx, pod); err != nil {
			return err
		}
	} else if pod.Status.Phase == corev1.PodFailed {
		var errorMessage string
		var kubeOneLogVar kubeOneLog
		var phaseError kubermaticv1.ExternalClusterPhase

		// fetch kubeone error code
		statusList := pod.Status.ContainerStatuses
		if len(statusList) > 0 {
			exitCode := statusList[0].State.Terminated.ExitCode
			phaseError = determineExitCode(exitCode)
		}

		logs, err := getPodLogs(ctx, pod)
		if err != nil {
			return err
		}

		logLines := strings.Split(logs, "\n")
		for _, logLine := range logLines {
			err = json.Unmarshal([]byte(logLine), &kubeOneLogVar)
			if err == nil {
				if kubeOneLogVar.Level == KubeOneLogLevelError {
					errorMessage = kubeOneLogVar.Message
				}
			}
		}

		log.Errorw("failed operation on kubeone cluster, for details check external cluster status", "operation", action)

		if err := r.updateClusterStatus(ctx, externalCluster, kubermaticv1.ExternalClusterCondition{
			Phase:   phaseError,
			Message: errorMessage,
		}); err != nil {
			return err
		}
		if err := r.Delete(ctx, pod); err != nil {
			return err
		}
	}

	return nil
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

func (r *reconciler) CreateOrUpdateKubeconfigSecretForCluster(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.ExternalCluster, kubeconfig, namespace string) (*providerconfig.GlobalSecretKeySelector, error) {
	kubeconfigRef, err := r.ensureKubeconfigSecret(ctx,
		log,
		cluster,
		map[string][]byte{
			resources.ExternalClusterKubeconfig: []byte(kubeconfig),
		}, namespace)
	if err != nil {
		return nil, err
	}
	return kubeconfigRef, nil
}

func (r *reconciler) ensureKubeconfigSecret(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.ExternalCluster, secretData map[string][]byte, namespace string) (*providerconfig.GlobalSecretKeySelector, error) {
	secretName := resources.KubeOneKubeconfigSecretName

	if cluster.Labels == nil {
		return nil, fmt.Errorf("missing cluster labels")
	}
	projectID, ok := cluster.Labels[kubermaticv1.ProjectIDLabelKey]
	if !ok {
		return nil, fmt.Errorf("missing cluster projectID label")
	}

	namespacedName := types.NamespacedName{Namespace: namespace, Name: secretName}
	existingSecret := &corev1.Secret{}

	if err := r.Get(ctx, namespacedName, existingSecret); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to probe for secret %q: %w", secretName, err)
		}
		return r.createKubeconfigSecret(ctx, log, secretData, secretName, projectID, namespace)
	}

	return updateKubeconfigSecret(ctx, r.Client, existingSecret, secretData, projectID, namespace)
}

func updateKubeconfigSecret(ctx context.Context, client ctrlruntimeclient.Client, existingSecret *corev1.Secret, secretData map[string][]byte, projectID, namespace string) (*providerconfig.GlobalSecretKeySelector, error) {
	if existingSecret.Data == nil {
		existingSecret.Data = map[string][]byte{}
	}

	requiresUpdate := false

	for k, v := range secretData {
		if !bytes.Equal(v, existingSecret.Data[k]) {
			requiresUpdate = true
			break
		}
	}

	if existingSecret.Labels == nil {
		existingSecret.Labels = map[string]string{kubermaticv1.ProjectIDLabelKey: projectID}
		requiresUpdate = true
	}

	if requiresUpdate {
		existingSecret.Data = secretData
		if err := client.Update(ctx, existingSecret); err != nil {
			return nil, fmt.Errorf("failed to update kubeconfig secret: %w", err)
		}
	}

	return &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      existingSecret.Name,
			Namespace: namespace,
		},
	}, nil
}

func (r *reconciler) createKubeconfigSecret(ctx context.Context, log *zap.SugaredLogger, secretData map[string][]byte, name, projectID, namespace string) (*providerconfig.GlobalSecretKeySelector, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{kubermaticv1.ProjectIDLabelKey: projectID},
		},
		Type: corev1.SecretTypeOpaque,
		Data: secretData,
	}
	if err := r.Create(ctx, secret); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create kubeconfig secret: %w", err)
		}
	}

	return &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      name,
			Namespace: namespace,
		},
	}, nil
}

func GenerateClient(cfg *clientcmdapi.Config) (ctrlruntimeclient.Client, error) {
	clientConfig, err := getRestConfig(cfg)
	if err != nil {
		return nil, err
	}
	restMapperCache := restmapper.New()
	client, err := restMapperCache.Client(clientConfig)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func getRestConfig(cfg *clientcmdapi.Config) (*rest.Config, error) {
	iconfig := clientcmd.NewNonInteractiveClientConfig(
		*cfg,
		"",
		&clientcmd.ConfigOverrides{},
		nil,
	)

	clientConfig, err := iconfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return clientConfig, nil
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

func (r *reconciler) getKubeOneSecret(ctx context.Context, ref providerconfig.GlobalSecretKeySelector) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: ref.Namespace, Name: ref.Name}, secret); err != nil {
		return nil, err
	}
	return secret, nil
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

func (r *reconciler) upgradeCluster(ctx context.Context, log *zap.SugaredLogger, externalCluster *kubermaticv1.ExternalCluster) (*corev1.Pod, error) {
	log.Info("Generating kubeone pod to upgrade kubeone...")
	generatedPod, err := r.generateKubeOneActionPod(ctx, log, externalCluster, UpgradeControlPlaneAction)
	if err != nil {
		return nil, err
	}

	log.Info("Creating kubeone pod to upgrade kubeone...")
	if err := r.Create(ctx, generatedPod); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, err
		}
	}
	log.Info("Waiting kubeone upgrade to complete...")

	return generatedPod, nil
}

func (r *reconciler) migrateCluster(ctx context.Context, log *zap.SugaredLogger, externalCluster *kubermaticv1.ExternalCluster) (*corev1.Pod, error) {
	log.Info("Generating kubeone pod to migrate kubeone...")
	generatedPod, err := r.generateKubeOneActionPod(ctx, log, externalCluster, MigrateContainerRuntimeAction)
	if err != nil {
		return nil, fmt.Errorf("could not generate kubeone pod %s/%s to migrate container runtime: %w", generatedPod.Name, generatedPod.Namespace, err)
	}

	log.Info("Creating kubeone pod to migrate kubeone...")
	if err := r.Create(ctx, generatedPod); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("could not create kubeone pod %s/%s to migrate kubeone cluster: %w", generatedPod.Name, generatedPod.Namespace, err)
		}
	}
	log.Info("Waiting kubeone container runtime migration to complete...")

	return generatedPod, nil
}

func (r *reconciler) generateKubeOneActionPod(ctx context.Context, log *zap.SugaredLogger, externalCluster *kubermaticv1.ExternalCluster, action string) (*corev1.Pod, error) {
	kubeOne := externalCluster.Spec.CloudSpec.KubeOne
	sshSecret, err := r.getKubeOneSecret(ctx, kubeOne.SSHReference)
	if err != nil {
		log.Errorw("could not find kubeone ssh secret", zap.Error(err))
		return nil, err
	}
	manifestSecret, err := r.getKubeOneSecret(ctx, kubeOne.ManifestReference)
	if err != nil {
		log.Errorw("could not find kubeone manifest secret", zap.Error(err))
		return nil, err
	}

	// kubeOneNamespace is the namespace where all resources are created for the kubeone cluster.
	kubeOneNamespace := manifestSecret.Namespace

	cm := generateConfigMap(kubeOneNamespace, action)
	if err := r.Create(ctx, cm); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create kubeone script configmap: %w", err)
		}
	}

	envVar := []corev1.EnvVar{}
	volumes := []corev1.Volume{}

	providerName := kubeOne.ProviderName
	if action == UpgradeControlPlaneAction || action == MigrateContainerRuntimeAction {
		credentialSecret, err := r.getKubeOneSecret(ctx, kubeOne.CredentialsReference)
		if err != nil {
			log.Errorw("could not find kubeone credential secret", zap.Error(err))
			return nil, err
		}
		envVar = setEnvForProvider(providerName, envVar, credentialSecret)
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

	var kubeonePodName, kubeoneCMName string

	switch {
	case action == ImportAction:
		kubeonePodName = KubeOneImportPod
		kubeoneCMName = KubeOneImportConfigMap
	case action == UpgradeControlPlaneAction:
		kubeonePodName = KubeOneUpgradePod
		kubeoneCMName = KubeOneUpgradeConfigMap
	case action == MigrateContainerRuntimeAction:
		kubeonePodName = KubeOneMigratePod
		kubeoneCMName = KubeOneMigrateConfigMap
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubeonePodName,
			Namespace: kubeOneNamespace,
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
					Image:   resources.KubeOneImage,
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
	}

	return pod, nil
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
