/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package envoymanager

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	envoyresourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"

	envoycachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	DefaultExposeAnnotationKey = "nodeport-proxy.k8s.io/expose"
	clusterConnectTimeout      = 1 * time.Second
)

type Options struct {
	Namespace           string
	ListenAddress       string
	EnvoyNodeName       string
	ExposeAnnotationKey string

	EnvoyStatsPort int
	EnvoyAdminPort int
}

type Reconciler struct {
	ctrlruntimeclient.Client
	Options

	Ctx context.Context
	Log *zap.SugaredLogger

	EnvoySnapshotCache envoycachev3.SnapshotCache
}

func (r *Reconciler) Reconcile(_ ctrl.Request) (ctrl.Result, error) {
	r.Log.Debug("Got reconcile request")
	err := r.sync()
	if err != nil {
		r.Log.Errorf("Failed to reconcile", zap.Error(err))
	}
	return ctrl.Result{}, err
}

func (r *Reconciler) sync() error {
	services := &corev1.ServiceList{}
	if err := r.List(r.Ctx, services,
		ctrlruntimeclient.InNamespace(r.Namespace),
		client.MatchingFields{r.ExposeAnnotationKey: "true"},
	); err != nil {
		return errors.Wrap(err, "failed to list service's")
	}

	listeners, clusters := r.makeInitialResources()

	for _, service := range services.Items {
		serviceKey := ServiceKey(&service)
		serviceLog := r.Log.With("service", serviceKey)
		// Only cover services which have the annotation: true
		if strings.ToLower(service.Annotations[r.ExposeAnnotationKey]) != "true" {
			serviceLog.Debugf("Skipping service: it does not have the annotation %s=true", r.ExposeAnnotationKey)
			continue
		}

		// We only manage NodePort services so Kubernetes takes care of allocating a unique port
		if service.Spec.Type != corev1.ServiceTypeNodePort {
			serviceLog.Warn("Skipping service: it is not of type NodePort", "service")
			return nil
		}

		eps := corev1.Endpoints{}
		if err := r.Get(context.Background(), types.NamespacedName{Namespace: service.Namespace, Name: service.Name}, &eps); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return errors.Wrap(err, fmt.Sprintf("failed to get endpoints for service '%s'", serviceKey))
		}
		// If we have no pods, dont bother creating a cluster.
		if len(eps.Subsets) == 0 {
			serviceLog.Debug("Skipping service: it has no running pods")
			continue
		}
		l, c := r.makeListenersAndClustersForService(&service, &eps)
		listeners = append(listeners, l...)
		clusters = append(clusters, c...)

	}

	// Get current snapshot
	currSnapshot, err := r.EnvoySnapshotCache.GetSnapshot(r.EnvoyNodeName)
	if err != nil {
		r.Log.Debugf("Setting first snapshot: %v", err)
		newSnapshot := envoycachev3.NewSnapshot(
			"v0.0.0",
			nil,       // endpoints
			clusters,  // clusters
			nil,       // routes
			listeners, // listeners
			nil,       // runtimes
			nil,       // secrets
		)
		if err := r.EnvoySnapshotCache.SetSnapshot(r.EnvoyNodeName, newSnapshot); err != nil {
			return errors.Wrap(err, "failed to set a new Envoy cache snapshot")
		}
		return nil
	}

	lastUsedVersion, err := semver.NewVersion(currSnapshot.GetVersion(envoyresourcev3.ClusterType))
	if err != nil {
		return errors.Wrap(err, "failed to parse version from last snapshot")
	}

	// Generate a new snapshot using the old version to be able to do a DeepEqual comparison
	snapshot := envoycachev3.NewSnapshot(
		lastUsedVersion.String(),
		nil,       // endpoints
		clusters,  // clusters
		nil,       // routes
		listeners, // listeners
		nil,       // runtimes
		nil,       // secrets
	)
	if reflect.DeepEqual(currSnapshot, snapshot) {
		r.Log.Debug("No changes detected")
		return nil
	}

	r.Log.Info("detected a change. Updating the Envoy config cache...")
	newVersion := lastUsedVersion.IncMajor()
	newSnapshot := envoycachev3.NewSnapshot(
		newVersion.String(),
		nil,       // endpoints
		clusters,  // clusters
		nil,       // routes
		listeners, // listeners
		nil,       // runtimes
		nil,       // secrets
	)

	if err := newSnapshot.Consistent(); err != nil {
		return errors.Wrap(err, "new Envoy config snapshot is not consistent")
	}

	if err := r.EnvoySnapshotCache.SetSnapshot(r.EnvoyNodeName, newSnapshot); err != nil {
		return errors.Wrap(err, "failed to set a new Envoy cache snapshot")
	}

	return nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Service{}, r.ExposeAnnotationKey, func(raw runtime.Object) []string {
		s := raw.(*corev1.Service)
		if s.ObjectMeta.Annotations == nil {
			return nil
		}
		return []string{strings.ToLower(s.ObjectMeta.Annotations[r.ExposeAnnotationKey])}
	}); err != nil {
		return fmt.Errorf("Error occurred while adding service index: %w", err)
	}
	return ctrl.NewControllerManagedBy(mgr).
		// Ensures that only one new Snapshot is generated at a time
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		For(&corev1.Service{}).
		Watches(&source.Kind{Type: &corev1.Endpoints{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(
				func(obj handler.MapObject) []ctrl.Request {
					return []ctrl.Request{{NamespacedName: types.NamespacedName{
						Name:      obj.Meta.GetName(),
						Namespace: obj.Meta.GetNamespace(),
					}}}
				})}).
		Complete(r)
}
