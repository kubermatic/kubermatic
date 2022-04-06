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

package encryption

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	k8cuserclusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	apiserverconfigv1 "k8s.io/apiserver/pkg/apis/config/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sigs.k8s.io/yaml"
)

const (
	ControllerName      = "kubermatic_encryption_controller"
	encryptionJobPrefix = "data-encryption"
)

// userClusterConnectionProvider offers functions to retrieve clients for the given user clusters
type userClusterConnectionProvider interface {
	GetClient(context.Context, *kubermaticv1.Cluster, ...k8cuserclusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

type Reconciler struct {
	ctrlruntimeclient.Client
	log                     *zap.SugaredLogger
	userClusterConnProvider userClusterConnectionProvider
	workerName              string
	recorder                record.EventRecorder

	versions kubermatic.Versions
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	userClusterConnProvider userClusterConnectionProvider,
	versions kubermatic.Versions) error {

	reconciler := &Reconciler{
		log:                     log.Named(ControllerName),
		Client:                  mgr.GetClient(),
		userClusterConnProvider: userClusterConnProvider,
		workerName:              workerName,

		recorder: mgr.GetEventRecorderFor(ControllerName),

		versions: versions,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return err
	}

	if err := c.Watch(
		&source.Kind{Type: &corev1.Secret{}},
		controllerutil.EnqueueClusterForNamespacedObject(mgr.GetClient()),
		predicateutil.ByName(resources.EncryptionConfigurationSecretName),
	); err != nil {
		return fmt.Errorf("failed to create watcher for corev1.Secret: %w", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &appsv1.Deployment{}},
		controllerutil.EnqueueClusterForNamespacedObject(mgr.GetClient()),
		predicateutil.ByName(resources.ApiserverDeploymentName),
	); err != nil {
		return fmt.Errorf("failed to create watcher for appsv1.Deployment: %w", err)
	}

	return c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}, predicateutil.Factory(func(o ctrlruntimeclient.Object) bool {
		cluster := o.(*kubermaticv1.Cluster)
		return cluster.Spec.Features[kubermaticv1.ClusterFeatureEncryptionAtRest]
	}))
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("Could not find cluster")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	log = log.With("cluster", cluster.Name)

	// return early if encryption is not enabled in spec and EncryptionIntialized condition is not set
	// TODO: make a predicate?
	if cluster.Spec.EncryptionConfiguration == nil || !cluster.Spec.EncryptionConfiguration.Enabled || cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionEncryptionInitialized, corev1.ConditionFalse) {
		return reconcile.Result{}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionClusterControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, log, cluster)
		},
	)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	if result == nil {
		result = &reconcile.Result{}
	}

	return *result, err

}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// wait for apiserver rollout to complete before updating status
	if ok, err := r.isApiserverUpdated(ctx, cluster); err != nil || !ok {
		log.Debugf("apiserver is not updated and ready yet, skipping...", err)
		return &reconcile.Result{}, err
	}
	// reconcile until encryption is successfully initialized
	if cluster.Spec.EncryptionConfiguration.Enabled && !cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionEncryptionInitialized, corev1.ConditionTrue) {
		log.Debug("EncryptionInitialized is not set yet, setting the condition ...")
		return r.setInitializedCondition(ctx, cluster)
	}

	primaryKey, err := r.getActiveKey(ctx, cluster)
	if err != nil {
		return &reconcile.Result{}, err
	}

	if cluster.Status.ActiveEncryptionKey != primaryKey {
		if err := kubermaticv1helper.UpdateClusterStatus(ctx, r.Client, cluster, func(c *kubermaticv1.Cluster) {
			c.Status.ActiveEncryptionKey = primaryKey
			kubermaticv1helper.SetClusterCondition(
				cluster,
				r.versions,
				kubermaticv1.ClusterConditionEncryptionFinished,
				corev1.ConditionFalse,
				"",
				fmt.Sprintf("Data re-encryption with key '%s' is pending", primaryKey),
			)
		}); err != nil {
			return &reconcile.Result{}, err
		}

		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionEncryptionFinished, corev1.ConditionFalse) {
		if ok, err := r.isApiserverUpdated(ctx, cluster); err != nil || !ok {
			return &reconcile.Result{RequeueAfter: 10 * time.Second}, err
		}

		return r.encryptData(ctx, log, cluster, primaryKey)
	}

	return &reconcile.Result{}, nil
}

func (r *Reconciler) setInitializedCondition(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// set the encryption initialized condition (should only happen once on every cluster)
	if err := kubermaticv1helper.UpdateClusterStatus(ctx, r.Client, cluster, func(c *kubermaticv1.Cluster) {
		kubermaticv1helper.SetClusterCondition(
			cluster,
			r.versions,
			kubermaticv1.ClusterConditionEncryptionInitialized,
			corev1.ConditionTrue,
			"",
			"Cluster data encryption has been initialized",
		)
	}); err != nil {
		return &reconcile.Result{}, err
	}

	return &reconcile.Result{}, nil
}

// TODO: create a better job that uses cmd/data-encryption-runner
func (r *Reconciler) encryptData(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, key string) (*reconcile.Result, error) {
	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{
		Name:      resources.EncryptionConfigurationSecretName,
		Namespace: cluster.Status.NamespaceName,
	}, &secret); err != nil {
		return &reconcile.Result{}, err
	}

	var jobList batchv1.JobList
	if err := r.List(ctx, &jobList, ctrlruntimeclient.MatchingLabels{
		"kubermatic.k8c.io/cluster":        cluster.Name,
		"kubermatic.k8c.io/encryption-key": strings.ReplaceAll(key, ":", "-"),
		"kubermatic.k8c.io/revision":       secret.ObjectMeta.ResourceVersion,
	}); err != nil {
		return &reconcile.Result{}, err
	} else {
		if len(jobList.Items) == 0 {
			job := batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: fmt.Sprintf("%s-%s", encryptionJobPrefix, cluster.Name),
					Namespace:    cluster.Status.NamespaceName,
					Labels: map[string]string{
						// TODO: do we have well-known labels aready?
						"kubermatic.k8c.io/cluster":        cluster.Name,
						"kubermatic.k8c.io/encryption-key": strings.ReplaceAll(key, ":", "-"),
						"kubermatic.k8c.io/revision":       secret.ObjectMeta.ResourceVersion,
					},
					// TODO: add owner reference?
				},
				Spec: batchv1.JobSpec{
					Parallelism:             pointer.Int32(1),
					Completions:             pointer.Int32(1),
					BackoffLimit:            pointer.Int32(0),
					TTLSecondsAfterFinished: pointer.Int32(86400),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyNever,
							Containers: []corev1.Container{
								{
									Name: "encryption-runner",
									// TODO: reconfigure image based on registry information
									Image:   "quay.io/kubermatic/util:2.0.0",
									Command: []string{"/bin/bash"},
									Args: []string{
										"-c",
										"kubectl get secrets --all-namespaces -o json | kubectl replace -f -"},
									Env: []corev1.EnvVar{
										{
											Name:  "KUBECONFIG",
											Value: "/opt/kubeconfig/kubeconfig",
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "kubeconfig",
											ReadOnly:  true,
											MountPath: "/opt/kubeconfig",
										},
									},
								},
							},
							SecurityContext: &corev1.PodSecurityContext{
								SeccompProfile: &corev1.SeccompProfile{
									Type: corev1.SeccompProfileTypeRuntimeDefault,
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "kubeconfig",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: resources.AdminKubeconfigSecretName,
										},
									},
								},
							},
						},
					},
				},
			}

			if err := r.Create(ctx, &job); err != nil {
				return &reconcile.Result{}, err
			}

			// we just created the job and need to check in with it later
			return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		} else {
			job := jobList.Items[0]
			if job.Status.Succeeded == 1 {
				if err := kubermaticv1helper.UpdateClusterStatus(ctx, r.Client, cluster, func(c *kubermaticv1.Cluster) {
					kubermaticv1helper.SetClusterCondition(
						cluster,
						r.versions,
						kubermaticv1.ClusterConditionEncryptionFinished,
						corev1.ConditionTrue,
						"",
						fmt.Sprintf("Data re-encryption with key '%s' finished", cluster.Status.ActiveEncryptionKey),
					)
				}); err != nil {
					return &reconcile.Result{}, err
				}

				return &reconcile.Result{}, nil
			} else if job.Status.Failed > 0 {
				// TODO: probably should handle failed encryption jobs differently
				if err := kubermaticv1helper.UpdateClusterStatus(ctx, r.Client, cluster, func(c *kubermaticv1.Cluster) {
					kubermaticv1helper.SetClusterCondition(
						cluster,
						r.versions,
						kubermaticv1.ClusterConditionEncryptionFinished,
						corev1.ConditionFalse,
						"JobFailed",
						fmt.Sprintf("Data re-encryption with key '%s' failed", cluster.Status.ActiveEncryptionKey),
					)
				}); err != nil {
					return &reconcile.Result{}, err
				}
				return &reconcile.Result{}, nil
			}
			return &reconcile.Result{}, nil
		}
	}
}

func (r *Reconciler) isApiserverUpdated(ctx context.Context, cluster *kubermaticv1.Cluster) (bool, error) {
	if cluster.Status.ExtendedHealth.Apiserver != kubermaticv1.HealthStatusUp {
		return false, nil
	}

	var deployment appsv1.Deployment
	if err := r.Client.Get(ctx, types.NamespacedName{
		Name:      resources.ApiserverDeploymentName,
		Namespace: cluster.Status.NamespaceName,
	}, &deployment); err != nil {
		return false, err
	}

	hasSecretVolume := false
	for _, volume := range deployment.Spec.Template.Spec.Volumes {
		if volume.Secret != nil && volume.Secret.SecretName == resources.EncryptionConfigurationSecretName {
			hasSecretVolume = true
		}
	}

	if !hasSecretVolume {
		return false, nil
	}

	if deployment.Status.Replicas != deployment.Status.UpdatedReplicas {
		return false, nil
	}

	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{
		Name:      resources.EncryptionConfigurationSecretName,
		Namespace: cluster.Status.NamespaceName,
	}, &secret); err != nil {
		return false, err
	}

	// TODO: replace hardcoded label
	if val, ok := deployment.Spec.Template.ObjectMeta.Labels["apiserver-encryption-configuration-secret-revision"]; !ok || val != secret.ObjectMeta.ResourceVersion {
		return false, nil
	}

	return true, nil
}

func (r *Reconciler) getActiveKey(ctx context.Context, cluster *kubermaticv1.Cluster) (string, error) {
	var (
		secret corev1.Secret
		config apiserverconfigv1.EncryptionConfiguration
	)
	if err := r.Get(ctx, types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: resources.EncryptionConfigurationSecretName}, &secret); err != nil {
		return "", err
	}

	if data, ok := secret.Data[resources.EncryptionConfigurationKeyName]; ok {
		if err := yaml.Unmarshal(data, &config); err != nil {
			return "", err
		}
	}

	keyName := config.Resources[0].Providers[0].Secretbox.Keys[0].Name

	return fmt.Sprintf("secretbox:%s", keyName), nil
}

func (r *Reconciler) updateCluster(ctx context.Context, cluster *kubermaticv1.Cluster, modify func(*kubermaticv1.Cluster), opts ...ctrlruntimeclient.MergeFromOption) error {
	oldCluster := cluster.DeepCopy()
	modify(cluster)
	if reflect.DeepEqual(oldCluster, cluster) {
		return nil
	}

	return r.Patch(ctx, cluster, ctrlruntimeclient.MergeFromWithOptions(oldCluster, opts...))
}
