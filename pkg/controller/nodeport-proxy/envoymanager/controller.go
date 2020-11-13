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
	"time"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	envoycachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	envoyresourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	DefaultExposeAnnotationKey = "nodeport-proxy.k8s.io/expose"
	clusterConnectTimeout      = 1 * time.Second
)

type Options struct {
	// Namespace where Services and Endpoints are watched.
	Namespace string
	// NodeName is the name used to retrieve the xds configuration.
	// It is supposed to match the id (AKA service-node) of the Envoy instance
	// being controlled.
	EnvoyNodeName string
	// ExposeAnnotationKey is the annotation used to expose services.
	ExposeAnnotationKey string

	// EnvoyAdminPort port used to exposed Envoy admin interface.
	EnvoyAdminPort int
	// EnvoyStatsPort is the port used to expose Envoy stats.
	EnvoyStatsPort int
}

type Reconciler struct {
	ctrlruntimeclient.Client
	Options

	Ctx context.Context
	Log *zap.SugaredLogger

	EnvoySnapshotCache envoycachev3.SnapshotCache
}

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	r.Log.Debugw("got reconcile request", "request", req)
	err := r.sync()
	if err != nil {
		r.Log.Errorf("failed to reconcile", zap.Error(err))
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
		if !isExposed(&service, r.ExposeAnnotationKey) {
			serviceLog.Debugf("skipping service: it does not have the annotation %s=true", r.ExposeAnnotationKey)
			continue
		}

		// We only manage NodePort services so Kubernetes takes care of allocating a unique port
		if service.Spec.Type != corev1.ServiceTypeNodePort {
			serviceLog.Warn("skipping service: it is not of type NodePort", "service")
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
			serviceLog.Debug("skipping service: it has no running pods")
			continue
		}
		l, c := r.makeListenersAndClustersForService(&service, &eps)
		listeners = append(listeners, l...)
		clusters = append(clusters, c...)

	}

	// Get current snapshot
	currSnapshot, err := r.EnvoySnapshotCache.GetSnapshot(r.EnvoyNodeName)
	if err != nil {
		r.Log.Debugf("setting first snapshot: %v", err)
		if err := r.EnvoySnapshotCache.SetSnapshot(r.EnvoyNodeName, newSnapshot("v0.0.0", clusters, listeners)); err != nil {
			return errors.Wrap(err, "failed to set a new Envoy cache snapshot")
		}
		return nil
	}

	lastUsedVersion, err := semver.NewVersion(currSnapshot.GetVersion(envoyresourcev3.ClusterType))
	if err != nil {
		return errors.Wrap(err, "failed to parse version from last snapshot")
	}

	// Generate a new snapshot using the old version to be able to do a DeepEqual comparison
	if reflect.DeepEqual(currSnapshot, newSnapshot(lastUsedVersion.String(), clusters, listeners)) {
		r.Log.Debug("no changes detected")
		return nil
	}

	r.Log.Info("detected a change. Updating the Envoy config cache...")
	newVersion := lastUsedVersion.IncMajor()
	newSnapshot := newSnapshot(newVersion.String(), clusters, listeners)

	if err := newSnapshot.Consistent(); err != nil {
		return errors.Wrap(err, "new Envoy config snapshot is not consistent")
	}

	if err := r.EnvoySnapshotCache.SetSnapshot(r.EnvoyNodeName, newSnapshot); err != nil {
		return errors.Wrap(err, "failed to set a new Envoy cache snapshot")
	}

	return nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(r.Ctx, &corev1.Service{}, r.ExposeAnnotationKey, func(raw runtime.Object) []string {
		var values []string
		svc := raw.(*corev1.Service)
		if isExposed(svc, r.ExposeAnnotationKey) {
			values = append(values, "ok")
		}
		return values
	}); err != nil {
		return fmt.Errorf("error occurred while adding service index: %w", err)
	}
	return ctrl.NewControllerManagedBy(mgr).
		// Ensures that only one new Snapshot is generated at a time
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		For(&corev1.Service{}, builder.WithPredicates(matchingAnnotationPredicate{annotation: r.ExposeAnnotationKey, log: r.Log})).
		Watches(&source.Kind{Type: &corev1.Endpoints{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(r.endpointsToService)}).
		Complete(r)
}

func (r *Reconciler) endpointsToService(obj handler.MapObject) []ctrl.Request {
	svcName := types.NamespacedName{
		Name:      obj.Meta.GetName(),
		Namespace: obj.Meta.GetNamespace(),
	}
	// Get the service associated to the Endpoints
	svc := corev1.Service{}
	if err := r.Client.Get(r.Ctx, svcName, &svc); err != nil {
		// Avoid enqueuing events for endpoints that do not have an associated
		// service (e.g. leader election).
		if !apierrors.IsNotFound(err) {
			r.Log.Errorw("error occurred while mapping endpoints to service", "endpoints", obj)
		}
		return nil
	}

	// Avoid enqueuing services that are not exposed.
	if !isExposed(&svc, r.ExposeAnnotationKey) {
		return nil
	}
	return []ctrl.Request{{NamespacedName: svcName}}
}

type matchingAnnotationPredicate struct {
	log        *zap.SugaredLogger
	annotation string
}

// Create returns true if the Create event should be processed
func (m matchingAnnotationPredicate) Create(e event.CreateEvent) bool {
	return m.match(e.Meta)
}

// Delete returns true if the Delete event should be processed
func (m matchingAnnotationPredicate) Delete(e event.DeleteEvent) bool {
	return m.match(e.Meta)
}

// Update returns true if the Update event should be processed
func (m matchingAnnotationPredicate) Update(e event.UpdateEvent) bool {
	return m.match(e.MetaNew)
}

// Generic returns true if the Generic event should be processed
func (m matchingAnnotationPredicate) Generic(e event.GenericEvent) bool {
	return m.match(e.Meta)
}

func (m matchingAnnotationPredicate) match(obj metav1.Object) bool {
	e := isExposed(obj, m.annotation)
	m.log.Debugw("processing event", "object", obj, "isExposed", e)
	return e
}
