package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"sort"

	"github.com/golang/glog"

	controllerutil "github.com/kubermatic/kubermatic/api/pkg/controller/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrltuntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const exposeAnnotationKey = "nodeport-proxy.k8s.io/expose"

var (
	lbName      string
	lbNamespace string
)

func main() {
	flag.StringVar(&lbName, "lb-name", "nodeport-lb", "name of the LoadBalancer service to manage.")
	flag.StringVar(&lbNamespace, "lb-namespace", "nodeport-proxy", "namespace of the LoadBalancer service to manage. Needs to exist")
	flag.Parse()

	config, err := ctrlruntimeconfig.GetConfig()
	if err != nil {
		glog.Fatalf("Failed to get config: %v", err)
	}

	stopCh := signals.SetupSignalHandler()

	mgr, err := manager.New(config, manager.Options{})
	if err != nil {
		glog.Fatalf("failed to construct mgr: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := &LBUpdater{
		ctx:         ctx,
		client:      mgr.GetClient(),
		lbNamespace: lbNamespace,
		lbName:      lbName,
	}

	ctrl, err := controller.New("lb-updater", mgr,
		controller.Options{Reconciler: r, MaxConcurrentReconciles: 1})
	if err != nil {
		glog.Fatalf("failed to construct controller: %v", err)
	}
	if err := ctrl.Watch(&source.Kind{Type: &corev1.Service{}}, controllerutil.EnqueueConst("")); err != nil {
		glog.Fatalf("Failed to add watch for Service: %v", err)
	}
	if err := mgr.Start(stopCh); err != nil {
		glog.Fatalf("manager ended: %v", err)
	}
}

// LBUpdater has all APIs to synchronize and updateLB the services and nodeports.
type LBUpdater struct {
	ctx    context.Context
	client ctrltuntimeclient.Client

	lbNamespace string
	lbName      string
}

func (u *LBUpdater) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	return reconcile.Result{}, u.syncLB(request.NamespacedName.String())
}

func (u *LBUpdater) syncLB(s string) error {
	glog.V(4).Infof("Syncing LB as Service %s got modified", s)

	services := &corev1.ServiceList{}
	if err := u.client.List(u.ctx, &ctrltuntimeclient.ListOptions{}, services); err != nil {
		return fmt.Errorf("failed to list services: %v", err)
	}

	var wantLBPorts []corev1.ServicePort
	for _, service := range services.Items {
		if service.Annotations[exposeAnnotationKey] != "true" {
			glog.V(4).Infof("skipping service %s/%s as the annotation %s is not set to 'true'", service.Namespace, service.Name, exposeAnnotationKey)
			continue
		}

		if service.Spec.ClusterIP == "" {
			glog.V(4).Infof("skipping service %s/%s as it has no clusterIP set", service.Namespace, service.Name)
			continue
		}

		for _, servicePort := range service.Spec.Ports {
			if servicePort.NodePort == 0 {
				glog.V(4).Infof("skipping service port %s/%s/%d as it has no clusterIP set", service.Namespace, service.Name, servicePort.NodePort)
				continue
			}
			wantLBPorts = append(wantLBPorts, corev1.ServicePort{
				Name:       fmt.Sprintf("%s-%s-%d-%d", service.Namespace, service.Name, servicePort.Port, servicePort.NodePort),
				Port:       servicePort.NodePort,
				TargetPort: intstr.FromInt(int(servicePort.NodePort)),
				Protocol:   corev1.ProtocolTCP,
			})
		}
	}

	if len(wantLBPorts) == 0 {
		// Nothing to do
		return nil
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

	wantLBPorts = fillWithNodePorts(wantLBPorts, lb.Spec.Ports)

	if !equality.Semantic.DeepEqual(wantLBPorts, lb.Spec.Ports) {
		glog.Infof("Updating LB ports...")
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
			buf.WriteString("\n")
		}
		buf.WriteString("======================\n")
		glog.Infof("%s\n", buf.String())
	}

	return nil
}

func fillWithNodePorts(wantPorts, lbPorts []corev1.ServicePort) []corev1.ServicePort {
	for wi := range wantPorts {
		for _, lp := range lbPorts {
			if wantPorts[wi].Name == lp.Name {
				wantPorts[wi].NodePort = lp.NodePort
			}
		}
	}

	return wantPorts
}
