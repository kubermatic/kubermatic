/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package webterminalsecretcleanup

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	controllerName = "kkp-webterminal-secret-cleanup"

	// minSecretAge gives a new session time to start its pod before the Secret is checked.
	minSecretAge = 15 * time.Minute
)

type reconciler struct {
	log    *zap.SugaredLogger
	client ctrlruntimeclient.Client
	// podReader reads pods without the cache, to avoid watching all pods in the cluster.
	podReader       ctrlruntimeclient.Reader
	clusterIsPaused userclustercontrollermanager.IsPausedChecker
	now             func() time.Time
}

func Add(log *zap.SugaredLogger, mgr manager.Manager, clusterIsPaused userclustercontrollermanager.IsPausedChecker) error {
	r := &reconciler{
		log:             log.Named(controllerName),
		client:          mgr.GetClient(),
		podReader:       mgr.GetAPIReader(),
		clusterIsPaused: clusterIsPaused,
		now:             time.Now,
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(controllerName).
		For(&corev1.Secret{}, builder.WithPredicates(
			predicateutil.ByNamespace(metav1.NamespaceSystem),
			predicateutil.Factory(func(o ctrlruntimeclient.Object) bool {
				return strings.HasPrefix(o.GetName(), resources.WebTerminalKubeconfigSecretPrefix)
			}),
		)).
		Build(r)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("secret", request.NamespacedName)
	log.Debug("Reconciling")

	if request.Namespace != metav1.NamespaceSystem {
		return reconcile.Result{}, nil
	}

	secret := &corev1.Secret{}
	if err := r.client.Get(ctx, request.NamespacedName, secret); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if secret.DeletionTimestamp != nil || !isWebTerminalSecret(secret) {
		return reconcile.Result{}, nil
	}

	if age := r.now().Sub(lastWritten(secret)); age < minSecretAge {
		log.Debugw("Web terminal Secret is too young to be removed, checking again later", "age", age)
		return reconcile.Result{RequeueAfter: minSecretAge - age}, nil
	}

	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		// unpausing causes no Secret event, so check again later
		return reconcile.Result{RequeueAfter: minSecretAge}, nil
	}

	mounted, err := r.isMountedByLiveTerminalPod(ctx, secret.Name)
	if err != nil {
		return reconcile.Result{}, err
	}
	if mounted {
		// deleting the pod causes no Secret event, so check again later
		log.Debug("Web terminal Secret is mounted by a terminal pod, checking again later")
		return reconcile.Result{RequeueAfter: minSecretAge}, nil
	}

	if err := r.client.Delete(ctx, secret); ctrlruntimeclient.IgnoreNotFound(err) != nil {
		return reconcile.Result{}, fmt.Errorf("failed to delete web terminal Secret: %w", err)
	}

	log.Info("Deleted leftover web terminal Secret")

	return reconcile.Result{}, nil
}

// isWebTerminalSecret returns true if the Secret has the web terminal label or its name
// is the md5 of its own user annotation.
func isWebTerminalSecret(secret *corev1.Secret) bool {
	if !strings.HasPrefix(secret.Name, resources.WebTerminalKubeconfigSecretPrefix) {
		return false
	}

	if secret.Labels[resources.AppLabelKey] == resources.WebTerminalAppLabelValue {
		return true
	}

	user := secret.Annotations[resources.WebTerminalUserAnnotationKey]
	if user == "" {
		return false
	}

	hash := md5.Sum([]byte(user))

	return secret.Name == resources.WebTerminalKubeconfigSecretPrefix+hex.EncodeToString(hash[:])
}

// lastWritten returns the newest managedFields time, or the creation time if there is none.
func lastWritten(secret *corev1.Secret) time.Time {
	last := secret.CreationTimestamp.Time
	for _, entry := range secret.ManagedFields {
		if entry.Time != nil && entry.Time.After(last) {
			last = entry.Time.Time
		}
	}

	return last
}

// isMountedByLiveTerminalPod returns true if a web terminal pod that is still running or starting
// and not being deleted mounts the Secret.
func (r *reconciler) isMountedByLiveTerminalPod(ctx context.Context, secretName string) (bool, error) {
	pods := &corev1.PodList{}
	if err := r.podReader.List(ctx, pods,
		ctrlruntimeclient.InNamespace(metav1.NamespaceSystem),
		ctrlruntimeclient.MatchingLabels{resources.AppLabelKey: resources.WebTerminalAppLabelValue},
	); err != nil {
		return false, fmt.Errorf("failed to list web terminal pods: %w", err)
	}

	for _, pod := range pods.Items {
		if pod.DeletionTimestamp != nil || pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
			continue
		}

		for _, volume := range pod.Spec.Volumes {
			if volume.Secret != nil && volume.Secret.SecretName == secretName {
				return true, nil
			}
		}
	}

	return false, nil
}
