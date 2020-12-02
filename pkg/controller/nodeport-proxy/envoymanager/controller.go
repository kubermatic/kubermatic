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

	"github.com/Masterminds/semver/v3"
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

	// EnvoySNIListenerPort is the port used by the SNI Listener.
	// When the value is less or equal than 0 the SNI Listener is disabled and
	// won't be configured in Envoy.
	EnvoySNIListenerPort int
	// EnvoyHTTP2ConnectListenerPort is the port used to listen for HTTP/2
	// CONNECT requests.
	// When the value is less or equal than 0 the HTTP/2 CONNECT Listener is
	// disabled and won't be configured in Envoy.
	EnvoyHTTP2ConnectListenerPort int
}

func (o Options) isSNIEnabled() bool {
	return o.EnvoySNIListenerPort > 0
}

func (o Options) isHTTP2ConnectEnabled() bool {
	return o.EnvoyHTTP2ConnectListenerPort > 0
}

// NewReconciler returns a new Reconciler or an error if something goes wrong
// during the initial snapshot setup.
func NewReconciler(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, opts Options) (*Reconciler, envoycachev3.SnapshotCache, error) {
	cache := envoycachev3.NewSnapshotCache(true, envoycachev3.IDHash{}, log)
	r := Reconciler{
		ctx:     ctx,
		log:     log,
		Client:  client,
		Options: opts,
		cache:   cache,
	}

	if err := r.cache.SetSnapshot(r.EnvoyNodeName, newSnapshotBuilder(log, portHostMappingFromAnnotation, opts).build("0.0.0")); err != nil {
		return nil, nil, errors.Wrap(err, "failed to set initial Envoy cache snapshot")
	}
	return &r, cache, nil
}

type Reconciler struct {
	ctx context.Context
	log *zap.SugaredLogger

	ctrlruntimeclient.Client
	Options

	cache envoycachev3.SnapshotCache
}

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	r.log.Debugw("got reconcile request", "request", req)
	err := r.sync()
	if err != nil {
		r.log.Errorf("failed to reconcile", zap.Error(err))
	}
	return ctrl.Result{}, err
}

func (r *Reconciler) sync() error {
	services := corev1.ServiceList{}
	if err := r.List(r.ctx, &services,
		ctrlruntimeclient.InNamespace(r.Namespace),
		client.MatchingFields{r.ExposeAnnotationKey: "true"},
	); err != nil {
		return errors.Wrap(err, "failed to list service's")
	}

	// Sort services in descending order by creation timestamp, in order to
	// skip newer services in case of 'hostname' conflict with SNI ExposeType.
	// Note that this is not fair, as the annotations may be changed during the
	// service lifetime. But this is a cheap solution and it is good enough for
	// the current needs.
	// A better option would be to use a CRD based approach and keeping sort
	// based on "expose" timestamp.
	SortServicesByCreationTimestamp(services.Items)

	sb := newSnapshotBuilder(r.log, portHostMappingFromAnnotation, r.Options)
	for _, service := range services.Items {
		svcKey := ServiceKey(&service)

		ets := extractExposeTypes(&service, r.ExposeAnnotationKey)

		// Get associated endpoints
		eps := corev1.Endpoints{}
		if err := r.Get(context.Background(), types.NamespacedName{Namespace: service.Namespace, Name: service.Name}, &eps); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return errors.Wrap(err, fmt.Sprintf("failed to get endpoints for service '%s'", svcKey))
		}
		// Add service to the service builder
		sb.addService(&service, &eps, ets)
	}

	// Get current snapshot
	currSnapshot, err := r.cache.GetSnapshot(r.EnvoyNodeName)
	if err != nil {
		r.log.Debugf("setting first snapshot: %v", err)
		if err := r.cache.SetSnapshot(r.EnvoyNodeName, sb.build("0.0.0")); err != nil {
			return errors.Wrap(err, "failed to set a new Envoy cache snapshot")
		}
		return nil
	}

	lastUsedVersion, err := semver.NewVersion(currSnapshot.GetVersion(envoyresourcev3.ClusterType))
	if err != nil {
		return errors.Wrap(err, "failed to parse version from last snapshot")
	}

	// Generate a new snapshot using the old version to be able to do a DeepEqual comparison
	if reflect.DeepEqual(currSnapshot, sb.build(lastUsedVersion.String())) {
		r.log.Debug("no changes detected")
		return nil
	}

	newVersion := lastUsedVersion.IncMajor()
	r.log.Infow("detected a change. Updating the Envoy config cache...", "version", newVersion.String())
	newSnapshot := sb.build(newVersion.String())

	if err := newSnapshot.Consistent(); err != nil {
		return errors.Wrap(err, "new Envoy config snapshot is not consistent")
	}

	if err := r.cache.SetSnapshot(r.EnvoyNodeName, newSnapshot); err != nil {
		return errors.Wrap(err, "failed to set a new Envoy cache snapshot")
	}

	return nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(r.ctx, &corev1.Service{}, r.ExposeAnnotationKey, func(raw runtime.Object) []string {
		svc := raw.(*corev1.Service)
		if isExposed(svc, r.ExposeAnnotationKey) {
			return []string{"true"}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("error occurred while adding service index: %w", err)
	}
	return ctrl.NewControllerManagedBy(mgr).
		// Ensures that only one new Snapshot is generated at a time
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		For(&corev1.Service{}, builder.WithPredicates(exposeAnnotationPredicate{annotation: r.ExposeAnnotationKey, log: r.log})).
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
	if err := r.Client.Get(r.ctx, svcName, &svc); err != nil {
		// Avoid enqueuing events for endpoints that do not have an associated
		// service (e.g. leader election).
		if !apierrors.IsNotFound(err) {
			r.log.Errorw("error occurred while mapping endpoints to service", "endpoints", obj)
		}
		return nil
	}

	// Avoid enqueuing events for services that are not exposed.
	if !isExposed(&svc, r.ExposeAnnotationKey) {
		return nil
	}
	return []ctrl.Request{{NamespacedName: svcName}}
}

// exposeAnnotationPredicate is used to filter out events associated to
// services that do not have the expose annotation and are thus not interesting
// for this controller.
type exposeAnnotationPredicate struct {
	log        *zap.SugaredLogger
	annotation string
}

// Create returns true if the Create event should be processed
func (e exposeAnnotationPredicate) Create(event event.CreateEvent) bool {
	return e.match(event.Meta)
}

// Delete returns true if the Delete event should be processed
func (e exposeAnnotationPredicate) Delete(event event.DeleteEvent) bool {
	return e.match(event.Meta)
}

// Update returns true if the Update event should be processed
func (e exposeAnnotationPredicate) Update(event event.UpdateEvent) bool {
	return e.match(event.MetaNew)
}

// Generic returns true if the Generic event should be processed
func (e exposeAnnotationPredicate) Generic(event event.GenericEvent) bool {
	return e.match(event.Meta)
}

func (e exposeAnnotationPredicate) match(obj metav1.Object) bool {
	ie := isExposed(obj, e.annotation)
	e.log.Debugw("processing event", "object", obj, "isExposed", ie)
	return ie
}
