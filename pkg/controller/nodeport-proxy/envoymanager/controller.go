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

	semverlib "github.com/Masterminds/semver/v3"
	"go.uber.org/zap"

	envoycachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	envoyresourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
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
	// EnvoyTunnelingListenerPort is the port used to listen for HTTP/2
	// CONNECT requests.
	// When the value is less or equal than 0 the HTTP/2 CONNECT Listener is
	// disabled and won't be configured in Envoy.
	EnvoyTunnelingListenerPort int

	// The following connection settings are optional Envoy overrides.
	// Zero values leave the corresponding Envoy fields unset.
	// SNIListenerIdleTimeout bounds how long inactive SNI listener downstream
	// TCP connections are kept open.
	SNIListenerIdleTimeout time.Duration
	// TunnelingConnectionIdleTimeout bounds how long inactive tunneling
	// listener downstream connections are kept open.
	TunnelingConnectionIdleTimeout time.Duration
	// TunnelingConnectionStreamTimeout bounds how long inactive HTTP/2 CONNECT
	// streams on the tunneling listener are kept open.
	TunnelingConnectionStreamTimeout time.Duration

	// DownstreamTCPKeepaliveTime configures idle time before downstream
	// listener sockets send TCP keepalive probes.
	DownstreamTCPKeepaliveTime time.Duration
	// DownstreamTCPKeepaliveInterval configures interval between downstream
	// listener keepalive probes.
	DownstreamTCPKeepaliveInterval time.Duration
	// DownstreamTCPKeepaliveProbes configures how many unanswered downstream
	// keepalive probes are allowed before the socket is considered dead.
	DownstreamTCPKeepaliveProbes int

	// UpstreamTCPKeepaliveTime configures idle time before upstream cluster
	// sockets send TCP keepalive probes.
	UpstreamTCPKeepaliveTime time.Duration
	// UpstreamTCPKeepaliveProbeInterval configures interval between upstream
	// keepalive probes.
	UpstreamTCPKeepaliveProbeInterval time.Duration
	// UpstreamTCPKeepaliveProbeAttempts configures how many unanswered upstream
	// keepalive probes are allowed before the socket is considered dead.
	UpstreamTCPKeepaliveProbeAttempts int
}

func (o Options) IsSNIEnabled() bool {
	return o.EnvoySNIListenerPort > 0
}

func (o Options) IsTunnelingEnabled() bool {
	return o.EnvoyTunnelingListenerPort > 0
}

func (o Options) HasSNIListenerIdleTimeout() bool {
	return o.SNIListenerIdleTimeout > 0
}

func (o Options) GetSNIListenerIdleTimeout() time.Duration {
	return o.SNIListenerIdleTimeout
}

func (o Options) HasTunnelingConnectionIdleTimeout() bool {
	return o.TunnelingConnectionIdleTimeout > 0
}

func (o Options) GetTunnelingConnectionIdleTimeout() time.Duration {
	return o.TunnelingConnectionIdleTimeout
}

func (o Options) HasTunnelingConnectionStreamTimeout() bool {
	return o.TunnelingConnectionStreamTimeout > 0
}

func (o Options) GetTunnelingConnectionStreamTimeout() time.Duration {
	return o.TunnelingConnectionStreamTimeout
}

func (o Options) HasDownstreamTCPKeepalive() bool {
	return o.DownstreamTCPKeepaliveTime > 0 || o.DownstreamTCPKeepaliveInterval > 0 || o.DownstreamTCPKeepaliveProbes > 0
}

func (o Options) GetDownstreamTCPKeepaliveTime() time.Duration {
	return o.DownstreamTCPKeepaliveTime
}

func (o Options) GetDownstreamTCPKeepaliveInterval() time.Duration {
	return o.DownstreamTCPKeepaliveInterval
}

func (o Options) GetDownstreamTCPKeepaliveProbes() uint32 {
	if o.DownstreamTCPKeepaliveProbes <= 0 {
		return 0
	}

	return uint32(o.DownstreamTCPKeepaliveProbes)
}

func (o Options) HasUpstreamTCPKeepalive() bool {
	return o.UpstreamTCPKeepaliveTime > 0 || o.UpstreamTCPKeepaliveProbeInterval > 0 || o.UpstreamTCPKeepaliveProbeAttempts > 0
}

func (o Options) GetUpstreamTCPKeepaliveTime() time.Duration {
	return o.UpstreamTCPKeepaliveTime
}

func (o Options) GetUpstreamTCPKeepaliveProbeInterval() time.Duration {
	return o.UpstreamTCPKeepaliveProbeInterval
}

func (o Options) GetUpstreamTCPKeepaliveProbeAttempts() uint32 {
	if o.UpstreamTCPKeepaliveProbeAttempts <= 0 {
		return 0
	}

	return uint32(o.UpstreamTCPKeepaliveProbeAttempts)
}

// NewReconciler returns a new Reconciler or an error if something goes wrong
// during the initial snapshot setup.
func NewReconciler(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, opts Options) (*Reconciler, envoycachev3.SnapshotCache, error) {
	cache := envoycachev3.NewSnapshotCache(true, envoycachev3.IDHash{}, log)
	r := Reconciler{
		options: opts,
		client:  client,
		log:     log,
		cache:   cache,
	}
	s, err := newSnapshotBuilder(log, portHostMappingFromAnnotation, opts).build("0.0.0")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build snapshot: %w", err)
	}

	if err := r.cache.SetSnapshot(ctx, r.options.EnvoyNodeName, s); err != nil {
		return nil, nil, fmt.Errorf("failed to set initial Envoy cache snapshot: %w", err)
	}
	return &r, cache, nil
}

type Reconciler struct {
	client  ctrlruntimeclient.Client
	log     *zap.SugaredLogger
	options Options
	cache   envoycachev3.SnapshotCache
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrlruntime.Request) (ctrlruntime.Result, error) {
	r.log.Debugw("got reconcile request", "request", req)
	err := r.sync(ctx)

	return ctrlruntime.Result{}, err
}

func (r *Reconciler) sync(ctx context.Context) error {
	services := corev1.ServiceList{}
	if err := r.client.List(ctx, &services,
		ctrlruntimeclient.InNamespace(r.options.Namespace),
		ctrlruntimeclient.MatchingFields{r.options.ExposeAnnotationKey: "true"},
	); err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	// Sort services in descending order by creation timestamp, in order to
	// skip newer services in case of 'hostname' conflict with SNI ExposeType.
	// Note that this is not fair, as the annotations may be changed during the
	// service lifetime. But this is a cheap solution and it is good enough for
	// the current needs.
	// A better option would be to use a CRD based approach and keeping sort
	// based on "expose" timestamp.
	SortServicesByCreationTimestamp(services.Items)

	sb := newSnapshotBuilder(r.log, portHostMappingFromAnnotation, r.options)
	for _, service := range services.Items {
		svcKey := ServiceKey(&service)

		ets := extractExposeTypes(&service, r.options.ExposeAnnotationKey)

		// Get associated endpoints
		eps := corev1.Endpoints{}
		if err := r.client.Get(ctx, types.NamespacedName{Namespace: service.Namespace, Name: service.Name}, &eps); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("failed to get endpoints for service '%s': %w", svcKey, err)
		}
		// Add service to the service builder
		sb.addService(&service, &eps, ets)
	}

	// Get current snapshot
	currSnapshot, err := r.cache.GetSnapshot(r.options.EnvoyNodeName)
	if err != nil {
		r.log.Debugf("setting first snapshot: %v", err)
		s, err := sb.build("0.0.0")
		if err != nil {
			return fmt.Errorf("failed to build the first snapshot: %w", err)
		}
		if err := r.cache.SetSnapshot(ctx, r.options.EnvoyNodeName, s); err != nil {
			return fmt.Errorf("failed to set a new Envoy cache snapshot: %w", err)
		}
		return nil
	}

	lastUsedVersion, err := semverlib.NewVersion(currSnapshot.GetVersion(envoyresourcev3.ClusterType))
	if err != nil {
		return fmt.Errorf("failed to parse version from last snapshot: %w", err)
	}

	s, err := sb.build(lastUsedVersion.String())
	if err != nil {
		return fmt.Errorf("failed to build snapshot: %w", err)
	}

	// Generate a new snapshot using the old version to be able to do a DeepEqual comparison
	if reflect.DeepEqual(currSnapshot, s) {
		r.log.Debug("no changes detected")
		return nil
	}

	newVersion := lastUsedVersion.IncMajor()
	r.log.Infow("detected a change. Updating the Envoy config cache...", "version", newVersion.String())
	newSnapshot, err := sb.build(newVersion.String())
	if err != nil {
		return fmt.Errorf("failed to build snapshot: %w", err)
	}

	if err := newSnapshot.Consistent(); err != nil {
		return fmt.Errorf("new Envoy config snapshot is not consistent: %w", err)
	}

	if err := r.cache.SetSnapshot(ctx, r.options.EnvoyNodeName, newSnapshot); err != nil {
		return fmt.Errorf("failed to set a new Envoy cache snapshot: %w", err)
	}

	return nil
}

func (r *Reconciler) SetupWithManager(ctx context.Context, mgr ctrlruntime.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &corev1.Service{}, r.options.ExposeAnnotationKey, func(raw ctrlruntimeclient.Object) []string {
		svc := raw.(*corev1.Service)
		if isExposed(svc, r.options.ExposeAnnotationKey) {
			return []string{"true"}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("error occurred while adding service index: %w", err)
	}
	return ctrlruntime.NewControllerManagedBy(mgr).
		// Ensures that only one new Snapshot is generated at a time
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		For(&corev1.Service{}, builder.WithPredicates(exposeAnnotationPredicate{annotation: r.options.ExposeAnnotationKey, log: r.log})).
		Watches(&corev1.Endpoints{}, handler.EnqueueRequestsFromMapFunc(r.newEndpointHandler())).
		Complete(r)
}

func (r *Reconciler) newEndpointHandler() handler.MapFunc {
	return func(ctx context.Context, obj ctrlruntimeclient.Object) []ctrlruntime.Request {
		svcName := types.NamespacedName{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
		}
		// Get the service associated to the Endpoints
		svc := corev1.Service{}
		if err := r.client.Get(ctx, svcName, &svc); err != nil {
			// Avoid enqueuing events for endpoints that do not have an associated
			// service (e.g. leader election).
			if !apierrors.IsNotFound(err) {
				r.log.Errorw("error occurred while mapping endpoints to service", "endpoints", obj)
			}
			return nil
		}

		// Avoid enqueuing events for services that are not exposed.
		if !isExposed(&svc, r.options.ExposeAnnotationKey) {
			return nil
		}
		return []ctrlruntime.Request{{NamespacedName: svcName}}
	}
}

// exposeAnnotationPredicate is used to filter out events associated to
// services that do not have the expose annotation and are thus not interesting
// for this controller.
type exposeAnnotationPredicate struct {
	log        *zap.SugaredLogger
	annotation string
}

// Create returns true if the Create event should be processed.
func (e exposeAnnotationPredicate) Create(event event.CreateEvent) bool {
	return e.match(event.Object)
}

// Delete returns true if the Delete event should be processed.
func (e exposeAnnotationPredicate) Delete(event event.DeleteEvent) bool {
	return e.match(event.Object)
}

// Update returns true if the Update event should be processed.
func (e exposeAnnotationPredicate) Update(event event.UpdateEvent) bool {
	return e.match(event.ObjectNew)
}

// Generic returns true if the Generic event should be processed.
func (e exposeAnnotationPredicate) Generic(event event.GenericEvent) bool {
	return e.match(event.Object)
}

func (e exposeAnnotationPredicate) match(obj metav1.Object) bool {
	ie := isExposed(obj, e.annotation)
	e.log.Debugw("processing event", "object", obj, "isExposed", ie)
	return ie
}
