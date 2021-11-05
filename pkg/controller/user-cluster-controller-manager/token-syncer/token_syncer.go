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

package tokensyncer

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"k8c.io/kubermatic/v2/pkg/resources"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"strings"

	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// This controller copies konnectivity service account token from user-cluster to seed-cluster
	// when the token updates.
	controllerName = "token_syncer_controller"
)

type reconciler struct {
	log             *zap.SugaredLogger
	client          ctrlruntimeclient.Client
	recorder        record.EventRecorder
	seedClient      ctrlruntimeclient.Client
	seedRecorder    record.EventRecorder
	clusterIsPaused userclustercontrollermanager.IsPausedChecker
	namespace       string
}

func Add(_ context.Context, log *zap.SugaredLogger, seedMgr, userMgr manager.Manager, namespace string,
	clusterIsPaused userclustercontrollermanager.IsPausedChecker) error {
	log = log.Named(controllerName)

	r := &reconciler{
		log:             log.With("controller", controllerName),
		client:          userMgr.GetClient(),
		recorder:        userMgr.GetEventRecorderFor(controllerName),
		seedClient:      seedMgr.GetClient(),
		seedRecorder:    seedMgr.GetEventRecorderFor(controllerName),
		clusterIsPaused: clusterIsPaused,
		namespace:       namespace,
	}
	c, err := controller.New(controllerName, userMgr, controller.Options{Reconciler: r})
	if err != nil {
		return fmt.Errorf("failed to create controller: %v", err)
	}

	pred := func(o ctrlruntimeclient.Object) bool {
		return o.GetNamespace() == "kube-system" &&
			strings.HasPrefix(o.GetName(), resources.KonnectivityServiceAccountName)
	}

	err = c.Watch(&source.Kind{Type: &corev1.Secret{}},
		&handler.EnqueueRequestForObject{},
		predicate.NewPredicateFuncs(pred),
	)

	if err != nil {
		return fmt.Errorf("failed to establish watch for the Namespace %v", err)
	}

	return nil
}

func (r *reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		return reconcile.Result{}, nil
	}

	// if not present it must be deleted ignore
	us := new(corev1.Secret)
	if err := r.client.Get(ctx, req.NamespacedName, us); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// check if present in seed cluster
	ss := new(corev1.Secret)
	if err := r.seedClient.Get(ctx, ctrlruntimeclient.ObjectKey{
		Namespace: r.namespace,
		Name:      resources.KonnectivityStolenAgentTokenSecretName,
	}, ss); err != nil {
		// if not present create it
		if kerrors.IsNotFound(err) {
			return r.createSecret(ctx, ss, us)
		}
		return reconcile.Result{}, err
	}

	// if present update it
	if resp, err := r.updateSecret(ctx, ss, us); err != nil {
		return resp, err
	}

	return reconcile.Result{}, nil
}

func (r *reconciler) updateSecret(ctx context.Context, ss *corev1.Secret, us *corev1.Secret) (reconcile.Result, error) {
	ss.Name = resources.KonnectivityStolenAgentTokenSecretName
	ss.Namespace = r.namespace

	caCrt, ok := us.Data["ca.crt"]
	if !ok {
		return reconcile.Result{}, fmt.Errorf("failed to get konnectivity-service-account secret token doesn't have ca.crt")
	}

	token, ok := us.Data["token"]
	if !ok {
		return reconcile.Result{}, fmt.Errorf("failed to get konnectivity-service-account secret token doesn't have token")
	}

	ss.Data = map[string][]byte{
		resources.KonnectivityStolenAgentTokenNameCert:  caCrt,
		resources.KonnectivityStolenAgentTokenNameToken: token,
	}

	if err := r.seedClient.Update(ctx, ss); err != nil {
		r.log.Errorf("failed to create secret %v", err)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *reconciler) createSecret(ctx context.Context, ss *corev1.Secret, us *corev1.Secret) (reconcile.Result, error) {
	ss.Name = resources.KonnectivityStolenAgentTokenSecretName
	ss.Namespace = r.namespace

	caCrt, ok := us.Data["ca.crt"]
	if !ok {
		return reconcile.Result{}, fmt.Errorf("failed to get konnectivity-service-account secret token doesn't have ca.crt")
	}

	token, ok := us.Data["token"]
	if !ok {
		return reconcile.Result{}, fmt.Errorf("failed to get konnectivity-service-account secret token doesn't have token")
	}

	ss.Data = map[string][]byte{
		resources.KonnectivityStolenAgentTokenNameCert:  caCrt,
		resources.KonnectivityStolenAgentTokenNameToken: token,
	}

	if err := r.seedClient.Create(ctx, ss); err != nil {
		r.log.Errorf("failed to create secret %v", err)
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}
