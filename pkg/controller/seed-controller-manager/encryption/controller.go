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
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	k8cuserclusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/resources"
	encryptionresources "k8c.io/kubermatic/v2/pkg/resources/encryption"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	apiserverconfigv1 "k8s.io/apiserver/pkg/apis/config/v1"
	"k8s.io/client-go/tools/record"
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
		kubermaticv1.ClusterConditionEncryptionControllerReconcilingSuccess,
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
	// reconcile until encryption is successfully initialized
	if cluster.Spec.EncryptionConfiguration.Enabled && !cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionEncryptionInitialized, corev1.ConditionTrue) {
		log.Debug("EncryptionInitialized is not set yet, setting initial encryption status and condition ...")
		return r.setInitializedCondition(ctx, cluster)
	}

	if cluster.Status.Encryption == nil {
		// TODO: handle this situation differently?
		return &reconcile.Result{}, nil
	}

	switch cluster.Status.Encryption.Phase {
	case kubermaticv1.ClusterEncryptionPhasePending:
		// TODO: in this phase, check health status. Is the secret up-to-date? is the apiserver up-to-date and healthy?
		// transition to ClusterEncryptionPhaseEncryptionNeeded or ClusterEncryptionPhaseActive.
		ok, err := r.isApiserverUpdated(ctx, cluster)
		if err != nil {
			return &reconcile.Result{}, err
		}

		if !ok {
			log.Debug("kube-apiserver is not using updated EncryptionConfiguration yet, retrying in 10s ...")
			return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}

		key, err := r.getActiveKey(ctx, cluster)
		if err != nil {
			return &reconcile.Result{}, err
		}

		if err := kubermaticv1helper.UpdateClusterStatus(ctx, r.Client, cluster, func(c *kubermaticv1.Cluster) {
			if c.Status.Encryption.ActiveKey != key {
				// the active key as per the parsed EncryptionConfiguration has changed; we need to re-run encryption
				c.Status.Encryption.Phase = kubermaticv1.ClusterEncryptionPhaseEncryptionNeeded
			} else {
				// EncryptionConfiguration was changed but the primary key did not change, so there is no need to re-run
				// encryption. We can skip right to ClusterEncryptionPhaseActive.
				c.Status.Encryption.Phase = kubermaticv1.ClusterEncryptionPhaseActive
			}
		}); err != nil {
			return &reconcile.Result{}, err
		}

		return &reconcile.Result{}, nil

	case kubermaticv1.ClusterEncryptionPhaseEncryptionNeeded:
		// TODO: in this phase, check for encryption Jobs and run one if it does not exist yet.
		// transition to ClusterEncryptionPhaseActive or ClusterEncryptionPhaseFailed.
		key, err := r.getActiveKey(ctx, cluster)
		if err != nil {
			return &reconcile.Result{}, err
		}

		return r.encryptData(ctx, log, cluster, key)

	case kubermaticv1.ClusterEncryptionPhaseActive:
		if cluster.Status.Encryption.ActiveKey != fmt.Sprintf("secretbox/%s", cluster.Spec.EncryptionConfiguration.Secretbox.Keys[0].Name) {
			if err := kubermaticv1helper.UpdateClusterStatus(ctx, r.Client, cluster, func(c *kubermaticv1.Cluster) {
				c.Status.Encryption.Phase = kubermaticv1.ClusterEncryptionPhasePending
			}); err != nil {
				return &reconcile.Result{}, err
			}
		}
		return &reconcile.Result{}, nil

	case kubermaticv1.ClusterEncryptionPhaseFailed:
		// TODO: how to recover from a failed encryption? Can you even recover?
		return &reconcile.Result{}, nil

	default:
		return &reconcile.Result{}, nil
	}
}

func (r *Reconciler) setInitializedCondition(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// set the encryption initialized condition (should only happen once on every cluster)
	if err := kubermaticv1helper.UpdateClusterStatus(ctx, r.Client, cluster, func(c *kubermaticv1.Cluster) {
		if cluster.Status.Encryption == nil {
			cluster.Status.Encryption = &kubermaticv1.ClusterEncryptionStatus{
				Phase: kubermaticv1.ClusterEncryptionPhasePending,
			}
		}

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
		"kubermatic.k8c.io/cluster":         cluster.Name,
		"kubermatic.k8c.io/secret-revision": secret.ObjectMeta.ResourceVersion,
	}); err != nil {
		return &reconcile.Result{}, err
	} else {
		if len(jobList.Items) == 0 {
			job := encryptionresources.EncryptionJob(cluster, &secret, key)

			if err := r.Create(ctx, &job); err != nil {
				return &reconcile.Result{}, err
			}

			// we just created the job and need to check in with it later
			return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		} else {
			job := jobList.Items[0]
			if job.Status.Succeeded == 1 {
				if err := kubermaticv1helper.UpdateClusterStatus(ctx, r.Client, cluster, func(c *kubermaticv1.Cluster) {
					cluster.Status.Encryption.Phase = kubermaticv1.ClusterEncryptionPhaseActive
					cluster.Status.Encryption.ActiveKey = key
				}); err != nil {
					return &reconcile.Result{}, err
				}

				return &reconcile.Result{}, nil
			} else if job.Status.Failed > 0 {
				if err := kubermaticv1helper.UpdateClusterStatus(ctx, r.Client, cluster, func(c *kubermaticv1.Cluster) {
					cluster.Status.Encryption.Phase = kubermaticv1.ClusterEncryptionPhaseFailed
				}); err != nil {
					return &reconcile.Result{}, err
				}
				return &reconcile.Result{}, nil
			}

			// no job result yet, requeue to read job status again in 10 seconds
			return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}
}

func (r *Reconciler) isApiserverUpdated(ctx context.Context, cluster *kubermaticv1.Cluster) (bool, error) {
	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{
		Name:      resources.EncryptionConfigurationSecretName,
		Namespace: cluster.Status.NamespaceName,
	}, &secret); err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	spec, err := json.Marshal(cluster.Spec.EncryptionConfiguration)
	if err != nil {
		return false, err
	}

	hash := sha1.New()
	hash.Write(spec)

	if val, ok := secret.ObjectMeta.Labels["kubermatic.k8c.io/encryption-spec-hash"]; !ok || val != hex.EncodeToString(hash.Sum(nil)) {
		// the secret on the cluster (or in the cache) doesn't seem updated yet
		return false, nil
	}

	var podList corev1.PodList
	if err := r.List(ctx, &podList, []ctrlruntimeclient.ListOption{
		ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName),
		ctrlruntimeclient.MatchingLabels{resources.AppLabelKey: "apiserver"},
	}...); err != nil {
		return false, err
	}

	if len(podList.Items) == 0 {
		return false, nil
	}

	for _, pod := range podList.Items {
		if val, ok := pod.Labels["apiserver-encryption-configuration-secret-revision"]; !ok || val != secret.ResourceVersion {
			return false, nil
		}
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

	return fmt.Sprintf("secretbox/%s", keyName), nil
}

func (r *Reconciler) updateCluster(ctx context.Context, cluster *kubermaticv1.Cluster, modify func(*kubermaticv1.Cluster), opts ...ctrlruntimeclient.MergeFromOption) error {
	oldCluster := cluster.DeepCopy()
	modify(cluster)
	if reflect.DeepEqual(oldCluster, cluster) {
		return nil
	}

	return r.Patch(ctx, cluster, ctrlruntimeclient.MergeFromWithOptions(oldCluster, opts...))
}
