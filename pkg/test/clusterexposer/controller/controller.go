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

package controller

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName                 = "cluster_exposer_controller"
	labelKey                       = "prow.k8s.io/id"
	serviceIdentifyerAnnotationKey = "clusterexposer/service-name"
)

type reconciler struct {
	log            *zap.SugaredLogger
	innerClient    ctrlruntimeclient.Client
	outerClient    ctrlruntimeclient.Client
	outerAPIReader ctrlruntimeclient.Reader
	jobID          string
}

func Add(log *zap.SugaredLogger, outer, inner manager.Manager, jobID string) error {
	log = log.Named(controllerName)

	r := &reconciler{
		log:            log,
		innerClient:    inner.GetClient(),
		outerClient:    outer.GetClient(),
		outerAPIReader: outer.GetAPIReader(),
		jobID:          jobID,
	}

	exposedServicesOnly := predicate.Factory(
		func(o ctrlruntimeclient.Object) bool {
			if _, exists := o.GetAnnotations()["nodeport-proxy.k8s.io/expose"]; exists {
				return true
			}
			return false
		},
	)

	outererServiceMapper := func(_ context.Context, a *corev1.Service) []reconcile.Request {
		val, exists := a.GetAnnotations()[serviceIdentifyerAnnotationKey]
		if !exists {
			return nil
		}
		split := strings.Split(val, "/")
		if n := len(split); n != 2 {
			log.Errorf("splitting value of key %q by `/` (%q) didn't yield two but %d results",
				serviceIdentifyerAnnotationKey, val, n)
			return nil
		}
		return []reconcile.Request{{
			NamespacedName: types.NamespacedName{Namespace: split[0], Name: split[1]},
		}}
	}

	outerServicePredicate := predicate.TypedFactory(func(o *corev1.Service) bool {
		return o.GetLabels()[labelKey] == jobID
	})

	_, err := builder.ControllerManagedBy(inner).
		Named(controllerName).
		For(&corev1.Service{}, builder.WithPredicates(exposedServicesOnly)).
		WatchesRawSource(source.Kind(
			outer.GetCache(),
			&corev1.Service{},
			handler.TypedEnqueueRequestsFromMapFunc(outererServiceMapper),
			outerServicePredicate,
		)).
		Build(r)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request.String())
	log.Debug("Reconciling")

	err := r.reconcile(ctx, log, request)

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, request reconcile.Request) error {
	innerService := &corev1.Service{}
	if err := r.innerClient.Get(ctx, request.NamespacedName, innerService); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("Got request for service that doesn't exist, returning")
			return nil
		}
		return fmt.Errorf("failed to get inner service: %w", err)
	}

	outerServices := &corev1.ServiceList{}
	labelSelector := ctrlruntimeclient.MatchingLabels(map[string]string{labelKey: r.jobID})
	if err := r.outerClient.List(ctx, outerServices, labelSelector); err != nil {
		return fmt.Errorf("failed to list service in outer cluster: %w", err)
	}
	outerService := getServiceFromServiceList(outerServices, request.NamespacedName)
	if outerService == nil {
		var err error
		outerService, err = r.createOuterService(ctx, request.String())
		if err != nil {
			return fmt.Errorf("failed to create service in outer cluster: %w", err)
		}
		log = log.With("outer-cluster-service-name", outerService.Name)
		log.Info("Successfully created service in outer cluster")
	}

	if n := len(outerService.Spec.Ports); n != 1 {
		return fmt.Errorf("expected outer service to have exactly one port, had %d", n)
	}
	if outerService.Spec.Type != corev1.ServiceTypeNodePort {
		return fmt.Errorf("expected outer service to be of type NodePort, was %q", outerService.Spec.Type)
	}
	if n := len(innerService.Spec.Ports); n != 1 {
		return fmt.Errorf("expected inner service to have exactly one port, had %d", n)
	}

	if innerService.Spec.Ports[0].NodePort == outerService.Spec.Ports[0].NodePort {
		log.Info("Node port already matched, nothing to do")
		return nil
	}

	log = log.With("nodeport", innerService.Spec.Ports[0].NodePort)
	log.Info("Updating nodeport of inner service")

	oldInnerService := innerService.DeepCopy()
	innerService.Spec.Ports[0].NodePort = outerService.Spec.Ports[0].NodePort
	if err := r.innerClient.Patch(ctx, innerService, ctrlruntimeclient.MergeFrom(oldInnerService)); err != nil {
		return fmt.Errorf("failed to update nodeport of service %s/%s to %d: %w", innerService.Namespace, innerService.Name, outerService.Spec.Ports[0].NodePort, err)
	}

	log.Info("Successfully updated nodeport of inner service")

	return nil
}

func (r *reconciler) createOuterService(ctx context.Context, targetServiceName string) (*corev1.Service, error) {
	newService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "cluster-exposer-",
			Namespace:    "default",
			Labels:       map[string]string{labelKey: r.jobID},
			Annotations:  map[string]string{serviceIdentifyerAnnotationKey: targetServiceName},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{labelKey: r.jobID},
			Type:     corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{
				{
					Name:     "secure",
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}
	myself := &corev1.Pod{}
	myselfName := types.NamespacedName{Namespace: "default", Name: r.jobID}
	// Use APIreader so we don't create a pod cache
	if err := r.outerAPIReader.Get(ctx, myselfName, myself); err != nil {
		return nil, fmt.Errorf("failed to get pod for self from outer cluster: %w", err)
	}
	if err := controllerutil.SetControllerReference(myself, newService, scheme.Scheme); err != nil {
		return nil, fmt.Errorf("failed to set owner ref for pod on outer service: %w", err)
	}
	if err := r.outerClient.Create(ctx, newService); err != nil {
		return nil, fmt.Errorf("failed to create outer service: %w", err)
	}
	// We must set our TargetPort to the same port as our NodePort
	oldSvc := newService.DeepCopy()
	newService.Spec.Ports[0].TargetPort = intstr.FromInt(int(newService.Spec.Ports[0].NodePort))
	if err := r.outerClient.Patch(ctx, newService, ctrlruntimeclient.MergeFrom(oldSvc)); err != nil {
		return nil, fmt.Errorf("failed to set target port to nodeport: %w", err)
	}
	return newService, nil
}

func getServiceFromServiceList(list *corev1.ServiceList, service types.NamespacedName) *corev1.Service {
	for _, svc := range list.Items {
		if svc.Annotations[serviceIdentifyerAnnotationKey] == service.String() {
			return &svc
		}
	}
	return nil
}
