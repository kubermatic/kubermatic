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

	"github.com/Azure/go-autorest/autorest/to"
	"go.uber.org/zap"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
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
)

const (
	// This controller manages kubeone external cluster.
	ControllerName = "kubeone_controller"
	ImportAction   = "import"
)

type reconciler struct {
	ctrlruntimeclient.Client
	log *zap.SugaredLogger
}

func Add(
	ctx context.Context,
	mgr manager.Manager,
	log *zap.SugaredLogger) error {
	reconciler := &reconciler{
		log:    log.Named(ControllerName),
		Client: mgr.GetClient(),
	}
	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler})
	if err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &kubermaticv1.ExternalCluster{}}, &handler.EnqueueRequestForObject{}, withEventFilter())
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
			return externalCluster.Spec.CloudSpec.KubeOne != nil && externalCluster.Spec.CloudSpec.KubeOne.ClusterStatus.Status == kubermaticv1.StatusProvisioning
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			obj := e.ObjectNew
			externalCluster, ok := obj.(*kubermaticv1.ExternalCluster)
			if !ok {
				return false
			}
			if externalCluster.Spec.CloudSpec == nil || externalCluster.Spec.CloudSpec.KubeOne == nil {
				return false
			}
			return externalCluster.Spec.CloudSpec.KubeOne.ClusterStatus.Status == kubermaticv1.StatusReconciling
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	externalCluster := &kubermaticv1.ExternalCluster{}
	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: metav1.NamespaceAll, Name: request.Name}, externalCluster); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("Could not find imported cluster")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	err := r.reconcile(ctx, externalCluster)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, externalCluster *kubermaticv1.ExternalCluster) error {
	kubeOne := externalCluster.Spec.CloudSpec.KubeOne

	if kubeOne.ClusterStatus.Status == kubermaticv1.StatusProvisioning {
		return r.importCluster(ctx, externalCluster)
	}

	return nil
}

func (r *reconciler) importCluster(ctx context.Context, externalCluster *kubermaticv1.ExternalCluster) error {
	r.log.Debugw("Importing kubeone cluster", "Cluster", externalCluster.Name)

	r.log.Debugw("Generate kubeone pod to fetch kubeconfig", "Cluster", externalCluster.Name)
	generatedPod, err := r.generateKubeOneKubeconfigPod(ctx, externalCluster, ImportAction)
	if err != nil {
		return fmt.Errorf("Could not generate kubeone pod for kubeone cluster %s, %w", externalCluster.Name, err)
	}

	r.log.Debugw("Create kubeone pod to fetch kubeconfig", "Cluster", externalCluster.Name)
	if err := r.Create(ctx, generatedPod); err != nil {
		return fmt.Errorf("Could not create kubeone pod for kubeone cluster %s, %w", externalCluster.Name, err)
	}
	pod := &corev1.Pod{}
	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: generatedPod.Namespace, Name: generatedPod.Name}, pod); err != nil {
		return err
	}
	for pod.Status.Phase != corev1.PodSucceeded {
		if pod.Status.Phase == corev1.PodFailed {
			importErr := fmt.Sprintf("Failed to import kubeone cluster %s, See Pod %s:%s logs for more details", externalCluster.Name, pod.Name, pod.Namespace)
			oldexternalCluster := externalCluster.DeepCopy()
			externalCluster.Spec.CloudSpec.KubeOne.ClusterStatus = kubermaticv1.KubeOneExternalClusterStatus{
				Status:        kubermaticv1.StatusError,
				StatusMessage: importErr,
			}
			if err := r.Patch(ctx, externalCluster, ctrlruntimeclient.MergeFrom(oldexternalCluster)); err != nil {
				return fmt.Errorf("failed to update kubeone cluster status %s: %w", externalCluster.Name, err)
			}
			return errors.New(importErr)
		}
		if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: generatedPod.Namespace, Name: generatedPod.Name}, pod); err != nil {
			return err
		}
	}

	config, err := getPodLogs(ctx, pod)
	if err != nil {
		return err
	}

	// cleanup pod as no longer required.
	err = r.Delete(ctx, pod)
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
	err = r.CreateOrUpdateKubeconfigSecretForCluster(ctx, externalCluster, config, generatedPod.Namespace)
	if err != nil {
		return err
	}
	r.log.Debugw("Kubeconfig reference created!", "Cluster", externalCluster.Name)
	// update kubeone externalcluster status.
	externalCluster.Spec.CloudSpec.KubeOne.ClusterStatus.Status = kubermaticv1.StatusRunning

	if err := r.Patch(ctx, externalCluster, ctrlruntimeclient.MergeFrom(oldexternalCluster)); err != nil {
		return fmt.Errorf("failed to add kubeconfig reference to %s: %w", externalCluster.Name, err)
	}

	r.log.Debugw("KubeOne Cluster Imported!", "Cluster", externalCluster.Name)
	return nil
}

func (r *reconciler) CreateOrUpdateKubeconfigSecretForCluster(ctx context.Context, cluster *kubermaticv1.ExternalCluster, kubeconfig, namespace string) error {
	kubeconfigRef, err := r.ensureKubeconfigSecret(ctx, cluster, map[string][]byte{
		resources.ExternalClusterKubeconfig: []byte(kubeconfig),
	}, namespace)
	if err != nil {
		return err
	}
	cluster.Spec.KubeconfigReference = kubeconfigRef
	return nil
}

func (r *reconciler) ensureKubeconfigSecret(ctx context.Context, cluster *kubermaticv1.ExternalCluster, secretData map[string][]byte, namespace string) (*providerconfig.GlobalSecretKeySelector, error) {
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
		return r.createKubeconfigSecret(ctx, secretData, secretName, projectID, namespace)
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

func (r *reconciler) createKubeconfigSecret(ctx context.Context, secretData map[string][]byte, name, projectID, namespace string) (*providerconfig.GlobalSecretKeySelector, error) {
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
			r.log.Debug("kubeone kubeconfig secret already exists")
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

func (r *reconciler) generateKubeOneKubeconfigPod(ctx context.Context, externalCluster *kubermaticv1.ExternalCluster, action string) (*corev1.Pod, error) {
	kubeOne := externalCluster.Spec.CloudSpec.KubeOne
	sshSecret, err := r.getKubeOneSecret(ctx, kubeOne.SSHReference)
	if err != nil {
		r.log.Errorf("Could not find kubeone ssh secret, %v", err)
		return nil, err
	}
	manifestSecret, err := r.getKubeOneSecret(ctx, kubeOne.ManifestReference)
	if err != nil {
		r.log.Errorf("Could not find kubeone manifest secret, %v", err)
		return nil, err
	}

	// kubeOneNamespace is the namespace where all resources are created for the kubeone cluster.
	kubeOneNamespace := manifestSecret.Namespace

	cm := generateConfigMap(resources.KubeOneScriptConfigMapName, kubeOneNamespace, action)
	if err := r.Create(ctx, cm); err != nil {
		if !kerrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create kubeone configmap: %w", err)
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
			GenerateName: "kubeone-",
			Namespace:    kubeOneNamespace,
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
	if action == ImportAction {
		scriptToRun = resources.KubeOneKubeConfigScript
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
