package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"sort"

	"github.com/go-test/deep"

	controllerutil "github.com/kubermatic/kubermatic/api/pkg/controller/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	defaultExposeAnnotationKey = "nodeport-proxy.k8s.io/expose"
	healthCheckPort            = 8002
)

var (
	lbName              string
	lbNamespace         string
	namespaced          bool
	exposeAnnotationKey string
)

func main() {
	klog.InitFlags(nil)
	flag.StringVar(&lbName, "lb-name", "nodeport-lb", "name of the LoadBalancer service to manage.")
	flag.StringVar(&lbNamespace, "lb-namespace", "nodeport-proxy", "namespace of the LoadBalancer service to manage. Needs to exist")
	flag.BoolVar(&namespaced, "namespaced", false, "Whether this controller should only watch services in the lbNamespace")
	flag.StringVar(&exposeAnnotationKey, "expose-annotation-key", defaultExposeAnnotationKey, "The annotation key used to determine if a Service should be exposed")
	flag.Parse()

	config, err := ctrlruntimeconfig.GetConfig()
	if err != nil {
		klog.Fatalf("Failed to get config: %v", err)
	}

	stopCh := signals.SetupSignalHandler()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-stopCh
		cancel()
	}()

	var namespace string
	if namespaced {
		namespace = lbNamespace
	}
	mgr, err := manager.New(config, manager.Options{Namespace: namespace})
	if err != nil {
		klog.Fatalf("failed to construct mgr: %v", err)
	}

	r := &LBUpdater{
		ctx:         ctx,
		client:      mgr.GetClient(),
		lbNamespace: lbNamespace,
		lbName:      lbName,
		namespace:   namespace,
	}

	ctrl, err := controller.New("lb-updater", mgr,
		controller.Options{Reconciler: r, MaxConcurrentReconciles: 1})
	if err != nil {
		klog.Fatalf("failed to construct controller: %v", err)
	}
	if err := ctrl.Watch(&source.Kind{Type: &corev1.Service{}}, controllerutil.EnqueueConst("")); err != nil {
		klog.Fatalf("Failed to add watch for Service: %v", err)
	}
	if err := mgr.Start(stopCh); err != nil {
		klog.Fatalf("manager ended: %v", err)
	}
}

// LBUpdater has all APIs to synchronize and updateLB the services and nodeports.
type LBUpdater struct {
	ctx    context.Context
	client ctrlruntimeclient.Client

	lbNamespace string
	lbName      string
	namespace   string
}

func (u *LBUpdater) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	err := u.syncLB(request.NamespacedName.String())
	if err != nil {
		klog.Errorf("Error syncing lb: %v", err)
	}
	return reconcile.Result{}, err
}

func (u *LBUpdater) syncLB(s string) error {
	klog.V(4).Infof("Syncing LB as Service %s got modified", s)

	services := &corev1.ServiceList{}
	opts := &ctrlruntimeclient.ListOptions{Namespace: u.namespace}
	if err := u.client.List(u.ctx, services, opts); err != nil {
		return fmt.Errorf("failed to list services: %v", err)
	}

	var wantLBPorts []corev1.ServicePort
	wantLBPorts = append(wantLBPorts, corev1.ServicePort{
		Name:       "healthz",
		Port:       healthCheckPort,
		TargetPort: intstr.FromInt(healthCheckPort),
		Protocol:   corev1.ProtocolTCP,
	})
	for _, service := range services.Items {
		if service.Annotations[exposeAnnotationKey] != "true" {
			klog.V(4).Infof("skipping service %s/%s as the annotation %s is not set to 'true'", service.Namespace, service.Name, exposeAnnotationKey)
			continue
		}

		if service.Spec.ClusterIP == "" {
			klog.V(4).Infof("skipping service %s/%s as it has no clusterIP set", service.Namespace, service.Name)
			continue
		}

		// We require a NodePort because we abuse it as allocation mechanism for a unique port
		for _, servicePort := range service.Spec.Ports {
			if servicePort.NodePort == 0 {
				klog.V(4).Infof("skipping service port %s/%s/%d as it has no nodePort set", service.Namespace, service.Name, servicePort.NodePort)
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
	if err := u.client.Get(u.ctx, types.NamespacedName{Namespace: u.lbNamespace, Name: u.lbName}, lb); err != nil {
		return fmt.Errorf("failed to get service %s/%s from lister: %v", u.lbNamespace, u.lbName, err)

	}

	//We need to sort both port list to be able to compare them for equality
	sort.Slice(wantLBPorts, func(i, j int) bool {
		return wantLBPorts[i].Name < wantLBPorts[j].Name
	})

	sort.Slice(lb.Spec.Ports, func(i, j int) bool {
		return lb.Spec.Ports[i].Name < lb.Spec.Ports[j].Name
	})

	wantLBPorts = fillNodePortsAndNames(wantLBPorts, lb.Spec.Ports)

	if !equality.Semantic.DeepEqual(wantLBPorts, lb.Spec.Ports) {
		diff := deep.Equal(wantLBPorts, lb.Spec.Ports)
		klog.Infof("Updating LB ports, diff: %v", diff)
		lb.Spec.Ports = wantLBPorts
		if err := u.client.Update(u.ctx, lb); err != nil {
			return fmt.Errorf("failed to update LB service %s/%s: %v", u.lbNamespace, u.lbName, err)
		}

		buf := &bytes.Buffer{}
		buf.WriteString("======================\n")
		buf.WriteString("Updated LB Ports:\n")
		for _, p := range lb.Spec.Ports {
			buf.WriteString(fmt.Sprintf("Name: %s\n", p.Name))
			buf.WriteString(fmt.Sprintf("Port: %d\n", p.Port))
			buf.WriteString(fmt.Sprintf("NodePort: %d\n", p.NodePort))
			buf.WriteString("\n")
		}
		buf.WriteString("======================\n")
		klog.Infof("%s\n", buf.String())
	} else {
		klog.V(4).Infof("LB service already up to date, nothing to do")
	}

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
	// needed because some LB implementations can not cope with a config change where only the
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
	if portToSet.Name != "healthz" {
		portToSet.Name = fmt.Sprintf("%s-%d", portToSet.Name, portToSet.Port)
	}
	// We must reset the NodePort, it is being abused to carry over the port of the target service
	// which is in all cases not a valid NodePort
	portToSet.NodePort = 0
}
