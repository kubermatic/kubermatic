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

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"sort"

	"github.com/go-logr/zapr"
	"github.com/go-test/deep"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/controller/nodeport-proxy/envoymanager"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/util/cli"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	healthCheckPort = 8002
)

var (
	lbName      string
	lbNamespace string
	namespaced  bool
)

func main() {
	logOpts := kubermaticlog.NewDefaultOptions()
	opts := envoymanager.Options{}
	logOpts.AddFlags(flag.CommandLine)

	flag.StringVar(&lbName, "lb-name", "nodeport-lb", "name of the LoadBalancer service to manage.")
	flag.StringVar(&lbNamespace, "lb-namespace", "nodeport-proxy", "namespace of the LoadBalancer service to manage. Needs to exist")
	flag.BoolVar(&namespaced, "namespaced", false, "Whether this controller should only watch services in the lbNamespace")
	flag.StringVar(&opts.ExposeAnnotationKey, "expose-annotation-key", nodeportproxy.DefaultExposeAnnotationKey, "The annotation key used to determine if a Service should be exposed")
	flag.IntVar(&opts.EnvoySNIListenerPort, "envoy-sni-port", 0, "Port used for SNI entry point.")
	flag.IntVar(&opts.EnvoyTunnelingListenerPort, "envoy-tunneling-port", 0, "Port used for HTTP/2 CONNECT termination.")
	flag.Parse()

	// setup signal handler
	ctx := signals.SetupSignalHandler()

	// init logging
	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar()

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))

	cli.Hello(log, "LoadBalancer Updater", nil)

	config, err := ctrlruntimeconfig.GetConfig()
	if err != nil {
		log.Fatalw("Failed to get config", zap.Error(err))
	}

	cacheOpts := cache.Options{
		DefaultNamespaces: map[string]cache.Config{},
	}

	namespace := ""
	if namespaced {
		namespace = lbNamespace
		cacheOpts.DefaultNamespaces[lbNamespace] = cache.Config{}
	}

	mgr, err := manager.New(config, manager.Options{
		Cache: cacheOpts,
	})
	if err != nil {
		log.Fatalw("Failed to construct mgr", zap.Error(err))
	}

	_, err = builder.ControllerManagedBy(mgr).
		Named("lb-updater").
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
		}).
		Watches(&corev1.Service{}, controllerutil.EnqueueConst("")).
		Build(&LBUpdater{
			client:      mgr.GetClient(),
			lbNamespace: lbNamespace,
			lbName:      lbName,
			namespace:   namespace,
			log:         log,
			opts:        opts,
		})
	if err != nil {
		log.Fatalw("Failed to construct controller", zap.Error(err))
	}

	if err := mgr.Start(ctx); err != nil {
		log.Fatalw("Manager ended", zap.Error(err))
	}
}

// LBUpdater has all APIs to synchronize and updateLB the services and nodeports.
type LBUpdater struct {
	client ctrlruntimeclient.Client

	lbNamespace string
	lbName      string
	namespace   string
	log         *zap.SugaredLogger
	opts        envoymanager.Options
}

func (u *LBUpdater) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	err := u.syncLB(ctx, request.String())
	if err != nil {
		u.log.Errorw("Error syncing LoadBalancer", zap.Error(err))
	}
	return reconcile.Result{}, err
}

func (u *LBUpdater) syncLB(ctx context.Context, s string) error {
	u.log.Debugw("Syncing LB because Service got modified", "service", s)

	services := &corev1.ServiceList{}
	opts := &ctrlruntimeclient.ListOptions{Namespace: u.namespace}
	if err := u.client.List(ctx, services, opts); err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	var wantLBPorts []corev1.ServicePort
	wantLBPorts = append(wantLBPorts, corev1.ServicePort{
		Name:       "healthz",
		Port:       healthCheckPort,
		TargetPort: intstr.FromInt(healthCheckPort),
		Protocol:   corev1.ProtocolTCP,
	})
	if u.opts.IsSNIEnabled() {
		wantLBPorts = append(wantLBPorts, corev1.ServicePort{
			Name:       "sni-listener",
			Port:       int32(u.opts.EnvoySNIListenerPort),
			TargetPort: intstr.FromInt(u.opts.EnvoySNIListenerPort),
			Protocol:   corev1.ProtocolTCP,
		})
	}
	if u.opts.IsTunnelingEnabled() {
		wantLBPorts = append(wantLBPorts, corev1.ServicePort{
			Name:       "tunneling-listener",
			Port:       int32(u.opts.EnvoyTunnelingListenerPort),
			TargetPort: intstr.FromInt(u.opts.EnvoyTunnelingListenerPort),
			Protocol:   corev1.ProtocolTCP,
		})
	}

	for _, service := range services.Items {
		serviceLog := u.log.With("namespace", service.Namespace).With("name", service.Name)

		if e := service.Annotations[u.opts.ExposeAnnotationKey]; e != "true" && e != nodeportproxy.NodePortType.String() {
			serviceLog.Debugw("Skipping service as the annotation is not set to 'true' or NodePort", "annotation", u.opts.ExposeAnnotationKey)
			continue
		}

		if service.Spec.ClusterIP == "" {
			serviceLog.Debug("Skipping service as it has no clusterIP set")
			continue
		}

		// We require a NodePort because we abuse it as allocation mechanism for a unique port
		for _, servicePort := range service.Spec.Ports {
			if servicePort.NodePort == 0 {
				serviceLog.Debugw("Skipping service port as it has no nodePort set", "port", servicePort.NodePort)
				continue
			}
			wantLBPorts = append(wantLBPorts, corev1.ServicePort{
				Name:       fmt.Sprintf("%s-%s", service.Namespace, service.Name),
				Port:       servicePort.NodePort,
				TargetPort: intstr.FromInt(int(servicePort.NodePort)),
				Protocol:   corev1.ProtocolTCP,
				// Not a mistake. We must know the original Port for name comparison and this is the only
				// field left in which we can put it.
				NodePort: servicePort.Port,
			})
		}
	}

	lb := &corev1.Service{}
	if err := u.client.Get(ctx, types.NamespacedName{Namespace: u.lbNamespace, Name: u.lbName}, lb); err != nil {
		return fmt.Errorf("failed to get service %s/%s from lister: %w", u.lbNamespace, u.lbName, err)
	}

	// We need to sort both port list to be able to compare them for equality
	sort.Slice(wantLBPorts, func(i, j int) bool {
		return wantLBPorts[i].Name < wantLBPorts[j].Name
	})

	sort.Slice(lb.Spec.Ports, func(i, j int) bool {
		return lb.Spec.Ports[i].Name < lb.Spec.Ports[j].Name
	})

	wantLBPorts = fillNodePortsAndNames(wantLBPorts, lb.Spec.Ports)

	if equality.Semantic.DeepEqual(wantLBPorts, lb.Spec.Ports) {
		u.log.Debug("LB service already up to date, nothing to do")
		return nil
	}

	diff := deep.Equal(wantLBPorts, lb.Spec.Ports)
	u.log.Debugw("Updating LB ports", "diff", diff)
	lb.Spec.Ports = wantLBPorts
	if err := u.client.Update(ctx, lb); err != nil {
		return fmt.Errorf("failed to update LB service %s/%s: %w", u.lbNamespace, u.lbName, err)
	}

	buf := &bytes.Buffer{}
	buf.WriteString("======================\n")
	buf.WriteString("Updated LB Ports:\n")
	for _, p := range lb.Spec.Ports {
		fmt.Fprintf(buf, "Name: %s\n", p.Name)
		fmt.Fprintf(buf, "Port: %d\n", p.Port)
		fmt.Fprintf(buf, "NodePort: %d\n\n", p.NodePort)
	}
	buf.WriteString("======================\n")
	u.log.Debug(buf.String())

	return nil
}

func fillNodePortsAndNames(wantPorts, lbPorts []corev1.ServicePort) []corev1.ServicePort {
	for wi := range wantPorts {
		setNodePortAndName(&wantPorts[wi], lbPorts)
	}

	return wantPorts
}

func setNodePortAndName(portToSet *corev1.ServicePort, lbPorts []corev1.ServicePort) {
	// We must support both the old name schema that included the port and resulted in a change
	// when the port was changed and the new one, where we only include the node port. This is
	// needed because some LB implementations cannot cope with a config change where only the
	// nodeport differs.
	// Additionally we have to compare the name directly, because in the case of the healthCheckPort
	// the NodePort or Port is not part of the name.
	oldSchemaName := fmt.Sprintf("%s-%d-%d", portToSet.Name, portToSet.NodePort, portToSet.Port)
	newSchemaName := fmt.Sprintf("%s-%d", portToSet.Name, portToSet.Port)
	for _, lbPort := range lbPorts {
		if oldSchemaName == lbPort.Name || newSchemaName == lbPort.Name || portToSet.Name == lbPort.Name {
			portToSet.Name = lbPort.Name
			portToSet.NodePort = lbPort.NodePort
			return
		}
	}
	if portToSet.Name != "healthz" && portToSet.Name != "sni-listener" && portToSet.Name != "tunneling-listener" {
		portToSet.Name = fmt.Sprintf("%s-%d", portToSet.Name, portToSet.Port)
	}
	// We must reset the NodePort, it is being abused to carry over the port of the target service
	// which is in all cases not a valid NodePort
	portToSet.NodePort = 0
}
