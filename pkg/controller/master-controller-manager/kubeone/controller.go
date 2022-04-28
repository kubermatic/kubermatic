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
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"go.uber.org/zap"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubeonev1beta2 "k8c.io/kubeone/pkg/apis/kubeone/v1beta2"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticpred "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/restmapper"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
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
	ControllerName = "kkp-kubeone-manager"

	// ImportAction is the action to import kubeone cluster.
	ImportAction = "import"

	// UpgradeControlPlaneAction is the action to upgrade kubeone cluster.
	UpgradeControlPlaneAction = "upgradeCP"

	// KubeOneUpgradePod is the name of kubeone pod performing upgrade.
	KubeOneUpgradePod = "kubeone-upgrade"

	// KubeOneKubeconfigPod is the name of kubeone pod fetching kubeconfig.
	KubeOneKubeconfigPod = "kubeone-kubeconfig"
)

type reconciler struct {
	ctrlruntimeclient.Client
	log *zap.SugaredLogger
	kubernetesprovider.ImpersonationClient
	secretKeySelector provider.SecretKeySelectorValueFunc
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
		createEventsOnly(),
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

func createEventsOnly() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func updateEventsOnly() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return true
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
			return false
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
	log.Debug("Processing")

	externalCluster := &kubermaticv1.ExternalCluster{}
	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: metav1.NamespaceAll, Name: request.Name}, externalCluster); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("Could not find imported cluster")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if externalCluster.DeletionTimestamp != nil {
		log.Debug("Deleting KubeOne Namespace")
		ns := &corev1.Namespace{}
		name := types.NamespacedName{Name: kubernetesprovider.GetKubeOneNamespaceName(externalCluster.Name)}
		if err := r.Get(ctx, name, ns); err != nil {
			if kerrors.IsNotFound(err) {
				log.Debug("Could not find external cluster namespace")
				return reconcile.Result{}, nil
			}
			return reconcile.Result{}, err
		}
		if err := r.Delete(ctx, ns); err != nil {
			return reconcile.Result{}, err
		}

		if err := kuberneteshelper.TryRemoveFinalizer(ctx, r, externalCluster, apiv1.ExternalClusterKubeOneNamespaceCleanupFinalizer); err != nil {
			log.Errorw("failed to remove kubeone namespace finalizer", zap.Error(err))
			return reconcile.Result{}, err
		}
	}
	return r.reconcile(ctx, log, externalCluster)
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, externalCluster *kubermaticv1.ExternalCluster) (reconcile.Result, error) {
	kubeconfigSecret := &corev1.Secret{}
	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: kubernetesprovider.GetKubeOneNamespaceName(externalCluster.Name), Name: resources.KubeOneKubeconfigSecretName}, kubeconfigSecret); err != nil {
		if kerrors.IsNotFound(err) {
			if err := r.importCluster(ctx, log, externalCluster); err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, err
	}
	externalClusterProvider, err := kubernetesprovider.NewExternalClusterProvider(r.ImpersonationClient, r.Client)
	if err != nil {
		return reconcile.Result{}, err
	}
	version, err := externalClusterProvider.GetVersion(ctx, externalCluster)
	if err != nil {
		return reconcile.Result{}, err
	}
	currentVersion := version.String()

	manifestSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: externalCluster.Spec.CloudSpec.KubeOne.ManifestReference.Namespace, Name: externalCluster.Spec.CloudSpec.KubeOne.ManifestReference.Name}, manifestSecret); err != nil {
		return reconcile.Result{}, fmt.Errorf("can not retrieve kubeone manifest secret: %w", err)
	}
	currentManifest := manifestSecret.Data[resources.KubeOneManifest]
	cluster := &kubeonev1beta2.KubeOneCluster{}
	if err := yaml.UnmarshalStrict(currentManifest, cluster); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to decode manifest secret data: %w", err)
	}

	wantedVersion := cluster.Versions.Kubernetes
	// check if a upgrade pod already exists
	pod := &corev1.Pod{}
	err = r.Get(ctx, ctrlruntimeclient.ObjectKey{
		Name:      KubeOneUpgradePod,
		Namespace: kubernetesprovider.GetKubeOneNamespaceName(externalCluster.Name),
	},
		pod,
	)
	if err != nil {
		if kerrors.IsNotFound(err) {
			if externalCluster.Spec.CloudSpec.KubeOne.ClusterStatus.Status != kubermaticv1.StatusError {
				if currentVersion != wantedVersion {
					log.Debugw("Upgrading kubeone cluster", "from", currentVersion, "to", wantedVersion)
					if err := r.upgradeCluster(ctx, log, externalCluster); err != nil {
						return reconcile.Result{}, err
					}
				}
			} else {
				return reconcile.Result{}, nil
			}
		} else {
			return reconcile.Result{}, err
		}
	} else {
		if pod.Status.Phase == corev1.PodPending || pod.Status.Phase == corev1.PodRunning {
			return reconcile.Result{RequeueAfter: time.Second * 10}, nil
		} else {
			err = r.checkPodStatus(ctx, log, pod, externalCluster)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	return reconcile.Result{}, nil
}

func (r *reconciler) checkPodStatus(ctx context.Context, log *zap.SugaredLogger, pod *corev1.Pod, externalCluster *kubermaticv1.ExternalCluster) error {
	log.Debugw("Checking kubeone pod status", "Pod", pod.Name)
	for pod.Status.Phase == corev1.PodSucceeded {
		oldexternalCluster := externalCluster.DeepCopy()
		// update kubeone externalcluster status.
		externalCluster.Spec.CloudSpec.KubeOne.ClusterStatus.Status = kubermaticv1.StatusRunning

		if err := r.Patch(ctx, externalCluster, ctrlruntimeclient.MergeFrom(oldexternalCluster)); err != nil {
			return fmt.Errorf("failed to add kubeconfig reference to %s: %w", externalCluster.Name, err)
		}
		log.Debug("KubeOne Cluster Upgraded!")
		if err := r.Delete(ctx, pod); err != nil {
			return err
		}
	}
	if pod.Status.Phase == corev1.PodFailed {
		upgradeErr := fmt.Sprintf("Failed to upgrade kubeone cluster %s", externalCluster.Name)
		oldexternalCluster := externalCluster.DeepCopy()
		externalCluster.Spec.CloudSpec.KubeOne.ClusterStatus = kubermaticv1.KubeOneExternalClusterStatus{
			Status:        kubermaticv1.StatusError,
			StatusMessage: upgradeErr,
		}
		if err := r.Patch(ctx, externalCluster, ctrlruntimeclient.MergeFrom(oldexternalCluster)); err != nil {
			return fmt.Errorf("failed to update kubeone cluster status %s: %w", externalCluster.Name, err)
		}
		log.Debug(upgradeErr)
		if err := r.Delete(ctx, pod); err != nil {
			return err
		}
	}

	return nil
}

func (r *reconciler) importCluster(ctx context.Context, log *zap.SugaredLogger, externalCluster *kubermaticv1.ExternalCluster) error {
	log.Debug("Importing kubeone cluster")

	log.Debug("Generate kubeone pod to fetch kubeconfig")
	generatedPod, err := r.generateKubeOneKubeconfigPod(ctx, log, externalCluster)
	if err != nil {
		return fmt.Errorf("Could not generate kubeone pod for kubeone cluster %s: %w", externalCluster.Name, err)
	}

	log.Debug("Create kubeone pod to fetch kubeconfig")
	if err := r.Create(ctx, generatedPod); err != nil {
		if !kerrors.IsAlreadyExists(err) {
			return fmt.Errorf("Could not create kubeone pod %s/%s for kubeone cluster %s: %w", KubeOneKubeconfigPod, generatedPod.Namespace, externalCluster.Name, err)
		}
	}

	for generatedPod.Status.Phase != corev1.PodSucceeded {
		if generatedPod.Status.Phase == corev1.PodFailed {
			importErr := fmt.Sprintf("Failed to import kubeone cluster %s, see Pod %s/%s logs for more details", externalCluster.Name, KubeOneKubeconfigPod, generatedPod.Namespace)
			oldexternalCluster := externalCluster.DeepCopy()
			externalCluster.Spec.CloudSpec.KubeOne.ClusterStatus = kubermaticv1.KubeOneExternalClusterStatus{
				Status:        kubermaticv1.StatusError,
				StatusMessage: importErr,
			}
			if err := r.Patch(ctx, externalCluster, ctrlruntimeclient.MergeFrom(oldexternalCluster)); err != nil {
				return fmt.Errorf("failed to update kubeone cluster status %s: %w", externalCluster.Name, err)
			}
			log.Debug(importErr)
			return nil
		}
		if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: generatedPod.Namespace, Name: KubeOneKubeconfigPod}, generatedPod); err != nil {
			return fmt.Errorf("failed to get kubeone kubeconfig pod for cluster %s: %w", externalCluster.Name, err)
		}
	}

	config, err := getPodLogs(ctx, generatedPod)
	if err != nil {
		return err
	}

	// cleanup pod as no longer required.
	err = r.Delete(ctx, generatedPod)
	if err != nil {
		return err
	}

	// cleanup configmap kubeone as no longer required.
	cm := &corev1.ConfigMap{}
	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: generatedPod.Namespace, Name: resources.KubeOneScriptConfigMapName}, cm); err != nil {
		return err
	}
	err = r.Delete(ctx, cm)
	if err != nil {
		return err
	}

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

	oldexternalCluster := externalCluster.DeepCopy()
	err = r.CreateOrUpdateKubeconfigSecretForCluster(ctx, log, externalCluster, config, generatedPod.Namespace)
	if err != nil {
		return err
	}
	log.Debug("Kubeconfig reference created!")
	// update kubeone externalcluster status.
	externalCluster.Spec.CloudSpec.KubeOne.ClusterStatus.Status = kubermaticv1.StatusRunning

	if err := r.Patch(ctx, externalCluster, ctrlruntimeclient.MergeFrom(oldexternalCluster)); err != nil {
		return fmt.Errorf("failed to add kubeconfig reference to %s: %w", externalCluster.Name, err)
	}

	log.Debug("KubeOne Cluster Imported!")
	return nil
}

func (r *reconciler) CreateOrUpdateKubeconfigSecretForCluster(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.ExternalCluster, kubeconfig, namespace string) error {
	kubeconfigRef, err := r.ensureKubeconfigSecret(ctx,
		log,
		cluster,
		map[string][]byte{
			resources.ExternalClusterKubeconfig: []byte(kubeconfig),
		}, namespace)
	if err != nil {
		return err
	}
	cluster.Spec.KubeconfigReference = kubeconfigRef
	return nil
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
		if !kerrors.IsNotFound(err) {
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
		if kerrors.IsAlreadyExists(err) {
			log.Debug("kubeone kubeconfig secret already exists")
		} else {
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
		return "", fmt.Errorf("creating client: %w", err)
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

func (r *reconciler) generateKubeOneKubeconfigPod(ctx context.Context, log *zap.SugaredLogger, externalCluster *kubermaticv1.ExternalCluster) (*corev1.Pod, error) {
	kubeOne := externalCluster.Spec.CloudSpec.KubeOne
	sshSecret, err := r.getKubeOneSecret(ctx, kubeOne.SSHReference)
	if err != nil {
		log.Errorw("Could not find kubeone ssh secret", zap.Error(err))
		return nil, err
	}
	manifestSecret, err := r.getKubeOneSecret(ctx, kubeOne.ManifestReference)
	if err != nil {
		log.Errorw("Could not find kubeone manifest secret", zap.Error(err))
		return nil, err
	}

	// kubeOneNamespace is the namespace where all resources are created for the kubeone cluster.
	kubeOneNamespace := manifestSecret.Namespace

	cm := generateConfigMap(resources.KubeOneScriptConfigMapName, kubeOneNamespace, ImportAction)
	if err := r.Create(ctx, cm); err != nil {
		if !kerrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create kubeone script configmap: %w", err)
		}
	}

	envVar := []corev1.EnvVar{}
	volumes := []corev1.Volume{}

	envVar = append(
		envVar,
		corev1.EnvVar{
			Name:  "PASSPHRASE",
			Value: string(sshSecret.Data["passphrase"]),
		},
	)

	envFrom := []corev1.EnvFromSource{}
	envFrom = append(
		envFrom,
		corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: sshSecret.Name,
				},
			},
		},
	)

	vm := []corev1.VolumeMount{}
	vm = append(
		vm,
		corev1.VolumeMount{
			Name:      "ssh-volume",
			MountPath: "/root/.ssh",
		},
		corev1.VolumeMount{
			Name:      "manifest-volume",
			MountPath: "/kubeonemanifest",
		},
		corev1.VolumeMount{
			Name:      "script-volume",
			MountPath: "/scripts",
		},
	)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KubeOneKubeconfigPod,
			Namespace: kubeOneNamespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "kubeone",
					Image:   resources.KubeOneImage,
					Command: []string{"/bin/sh"},
					Args: []string{
						"-c",
						"./scripts/script.sh",
					},
					EnvFrom:      envFrom,
					Env:          envVar,
					Resources:    corev1.ResourceRequirements{},
					VolumeMounts: vm,
				},
			},
			Volumes: append(
				volumes,
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
							DefaultMode: to.Int32Ptr(256),
							SecretName:  sshSecret.Name,
						},
					},
				},
				corev1.Volume{
					Name: "script-volume",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "kubeone",
							},
							DefaultMode: to.Int32Ptr(448),
						},
					},
				},
			),
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	return pod, nil
}

func generateConfigMap(name, namespace, action string) *corev1.ConfigMap {
	var scriptToRun string
	scriptToRun = resources.KubeOneScript
	if action == ImportAction {
		scriptToRun += "kubeone kubeconfig -m kubeonemanifest/manifest"
	}

	if action == UpgradeControlPlaneAction {
		scriptToRun += "kubeone apply -m kubeonemanifest/manifest -y"
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

func (r *reconciler) upgradeCluster(ctx context.Context, log *zap.SugaredLogger, externalCluster *kubermaticv1.ExternalCluster) error {
	log.Debug("Generate kubeone pod to upgrade kubeone")
	generatedPod, err := r.generateKubeOneUpgradePod(ctx, log, externalCluster)
	if err != nil {
		return fmt.Errorf("Could not generate kubeone pod for kubeone cluster upgrade %s: %w", externalCluster.Name, err)
	}

	log.Debug("Create kubeone pod to upgrade kubeone")
	if err := r.Create(ctx, generatedPod); err != nil {
		if !kerrors.IsAlreadyExists(err) {
			return fmt.Errorf("Could not create kubeone pod for kubeone cluster upgrade %s: %w", externalCluster.Name, err)
		}
	}
	log.Debug("Waiting kubeone upgrade to complete...")
	oldexternalCluster := externalCluster.DeepCopy()
	// update kubeone externalcluster status.
	externalCluster.Spec.CloudSpec.KubeOne.ClusterStatus.Status = kubermaticv1.StatusReconciling

	if err := r.Patch(ctx, externalCluster, ctrlruntimeclient.MergeFrom(oldexternalCluster)); err != nil {
		return fmt.Errorf("failed to add kubeconfig reference to %s: %w", externalCluster.Name, err)
	}

	return nil
}

func (r *reconciler) generateKubeOneUpgradePod(ctx context.Context, log *zap.SugaredLogger, externalCluster *kubermaticv1.ExternalCluster) (*corev1.Pod, error) {
	kubeOne := externalCluster.Spec.CloudSpec.KubeOne
	sshSecret, err := r.getKubeOneSecret(ctx, kubeOne.SSHReference)
	if err != nil {
		log.Errorw("Could not find kubeone ssh secret", zap.Error(err))
		return nil, err
	}
	manifestSecret, err := r.getKubeOneSecret(ctx, kubeOne.ManifestReference)
	if err != nil {
		log.Errorw("Could not find kubeone manifest secret", zap.Error(err))
		return nil, err
	}
	credentialSecret, err := r.getKubeOneSecret(ctx, kubeOne.CredentialsReference)
	if err != nil {
		log.Errorw("Could not find kubeone credential secret", zap.Error(err))
		return nil, err
	}

	// kubeOneNamespace is the namespace where all resources are created for the kubeone cluster.
	kubeOneNamespace := manifestSecret.Namespace

	cm := generateConfigMap(resources.KubeOneScriptConfigMapName, kubeOneNamespace, UpgradeControlPlaneAction)
	if err := r.Create(ctx, cm); err != nil {
		if !kerrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create kubeone script configmap: %w", err)
		}
	}

	envVar := []corev1.EnvVar{}
	volumes := []corev1.Volume{}

	providerName := kubeOne.ProviderName
	envVar = setEnvForProvider(providerName, envVar, credentialSecret)

	envVar = append(
		envVar,
		corev1.EnvVar{
			Name:  "PASSPHRASE",
			Value: string(sshSecret.Data["passphrase"]),
		},
	)

	envFrom := []corev1.EnvFromSource{}
	envFrom = append(
		envFrom,
		corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: credentialSecret.Name,
				},
			},
		},
		corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: sshSecret.Name,
				},
			},
		},
	)

	vm := []corev1.VolumeMount{}
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

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KubeOneUpgradePod,
			Namespace: kubeOneNamespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       externalCluster.Name,
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.ExternalClusterKind,
					Controller: to.BoolPtr(true),
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
					EnvFrom:      envFrom,
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
							DefaultMode: to.Int32Ptr(256),
							SecretName:  sshSecret.Name,
						},
					},
				},
				corev1.Volume{
					Name: "script-volume",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "kubeone",
							},
							DefaultMode: to.Int32Ptr(448),
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
	if providerName == resources.KubeOneAWS {
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:  "AWS_ACCESS_KEY_ID",
				Value: string(credentialSecret.Data[resources.AWSAccessKeyID]),
			},
			corev1.EnvVar{
				Name:  "AWS_SECRET_ACCESS_KEY",
				Value: string(credentialSecret.Data[resources.AWSSecretAccessKey]),
			},
		)
	}
	if providerName == resources.KubeOneAzure {
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:  "ARM_CLIENT_ID",
				Value: string(credentialSecret.Data[resources.AzureClientID]),
			},
			corev1.EnvVar{
				Name:  "ARM_CLIENT_SECRET",
				Value: string(credentialSecret.Data[resources.AzureClientSecret]),
			},
			corev1.EnvVar{
				Name:  "ARM_TENANT_ID",
				Value: string(credentialSecret.Data[resources.AzureTenantID]),
			},
			corev1.EnvVar{
				Name:  "ARM_SUBSCRIPTION_ID",
				Value: string(credentialSecret.Data[resources.AzureSubscriptionID]),
			},
		)
	}
	if providerName == resources.KubeOneDigitalOcean {
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:  "DIGITALOCEAN_TOKEN",
				Value: string(credentialSecret.Data[resources.DigitaloceanToken]),
			},
		)
	}
	if providerName == resources.KubeOneGCP {
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:  "GOOGLE_CREDENTIALS",
				Value: string(credentialSecret.Data[resources.GCPServiceAccount]),
			},
		)
	}
	if providerName == resources.KubeOneHetzner {
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:  "HCLOUD_TOKEN",
				Value: string(credentialSecret.Data[resources.HetznerToken]),
			},
		)
	}
	if providerName == resources.KubeOneNutanix {
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:  "NUTANIX_ENDPOINT",
				Value: string(credentialSecret.Data[resources.NutanixEndpoint]),
			},
			corev1.EnvVar{
				Name:  "NUTANIX_PORT",
				Value: string(credentialSecret.Data[resources.NutanixPort]),
			},
			corev1.EnvVar{
				Name:  "NUTANIX_USERNAME",
				Value: string(credentialSecret.Data[resources.NutanixUsername]),
			},
			corev1.EnvVar{
				Name:  "NUTANIX_PASSWORD",
				Value: string(credentialSecret.Data[resources.NutanixPassword]),
			},
			corev1.EnvVar{
				Name:  "NUTANIX_PE_ENDPOINT",
				Value: string(credentialSecret.Data[resources.NutanixCSIEndpoint]),
			},
			corev1.EnvVar{
				Name:  "NUTANIX_PE_USERNAME",
				Value: string(credentialSecret.Data[resources.NutanixCSIUsername]),
			},
			corev1.EnvVar{
				Name:  "NUTANIX_PE_PASSWORD",
				Value: string(credentialSecret.Data[resources.NutanixCSIPassword]),
			},
			corev1.EnvVar{
				Name:  "NUTANIX_INSECURE",
				Value: string(credentialSecret.Data[resources.NutanixAllowInsecure]),
			},
			corev1.EnvVar{
				Name:  "NUTANIX_PROXY_URL",
				Value: string(credentialSecret.Data[resources.NutanixProxyURL]),
			},
			corev1.EnvVar{
				Name:  "NUTANIX_CLUSTER_NAME",
				Value: string(credentialSecret.Data[resources.NutanixClusterName]),
			},
		)
	}
	if providerName == resources.KubeOneOpenStack {
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:  "OS_AUTH_URL",
				Value: string(credentialSecret.Data[resources.OpenstackAuthURL]),
			},
			corev1.EnvVar{
				Name:  "OS_USERNAME",
				Value: string(credentialSecret.Data[resources.OpenstackUsername]),
			},
			corev1.EnvVar{
				Name:  "OS_PASSWORD",
				Value: string(credentialSecret.Data[resources.OpenstackPassword]),
			},
			corev1.EnvVar{
				Name:  "OS_REGION_NAME",
				Value: string(credentialSecret.Data[resources.OpenstackRegion]),
			},
			corev1.EnvVar{
				Name:  "OS_DOMAIN_NAME",
				Value: string(credentialSecret.Data[resources.OpenstackDomain]),
			},
			corev1.EnvVar{
				Name:  "OS_TENANT_ID",
				Value: string(credentialSecret.Data[resources.OpenstackTenantID]),
			},
			corev1.EnvVar{
				Name:  "OS_TENANT_NAME",
				Value: string(credentialSecret.Data[resources.OpenstackTenant]),
			},
		)
	}
	if providerName == resources.KubeOneEquinix {
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:  "PACKET_AUTH_TOKEN",
				Value: string(credentialSecret.Data[resources.PacketAPIKey]),
			},
			corev1.EnvVar{
				Name:  "PACKET_PROJECT_ID",
				Value: string(credentialSecret.Data[resources.PacketProjectID]),
			},
		)
	}
	if providerName == resources.KubeOneVSphere {
		envVar = append(
			envVar,
			corev1.EnvVar{
				Name:  "VSPHERE_SERVER",
				Value: string(credentialSecret.Data[resources.VsphereServer]),
			},
			corev1.EnvVar{
				Name:  "VSPHERE_USER",
				Value: string(credentialSecret.Data[resources.VsphereUsername]),
			},
			corev1.EnvVar{
				Name:  "VSPHERE_PASSWORD",
				Value: string(credentialSecret.Data[resources.VspherePassword]),
			},
		)
	}

	return envVar
}
