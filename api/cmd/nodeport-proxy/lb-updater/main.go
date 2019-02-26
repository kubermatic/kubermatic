package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/oklog/run"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corev1lister "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

const exposeAnnotationKey = "nodeport-proxy.k8s.io/expose"

var (
	kubeconfig  string
	master      string
	lbName      string
	lbNamespace string
)

func main() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	flag.StringVar(&master, "master", "", "master url")
	flag.StringVar(&lbName, "lb-name", "nodeport-lb", "name of the LoadBalancer service to manage.")
	flag.StringVar(&lbNamespace, "lb-namespace", "nodeport-proxy", "namespace of the LoadBalancer service to manage. Needs to exist")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatal(err)
	}

	kubeInformerFactory := coreinformers.NewSharedInformerFactory(client, 30*time.Minute)

	u := LBUpdater{
		client: client,
		lister: kubeInformerFactory.Core().V1().Services().Lister(),
		queue:  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "services"),

		lbNamespace: lbNamespace,
		lbName:      lbName,
	}

	kubeInformerFactory.Core().V1().Services().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { u.enqueue(obj.(*corev1.Service)) },
		UpdateFunc: func(oldObj, newObj interface{}) { u.enqueue(newObj.(*corev1.Service)) },
		DeleteFunc: func(obj interface{}) {
			s, ok := obj.(*corev1.Service)
			if !ok {
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					runtime.HandleError(fmt.Errorf("couldn't get object from tombstone %#v", obj))
					return
				}
				s, ok = tombstone.Obj.(*corev1.Service)
				if !ok {
					runtime.HandleError(fmt.Errorf("tombstone contained object that is not a Service %#v", obj))
					return
				}
			}
			u.enqueue(s)
		},
	})

	kubeInformerFactory.Start(wait.NeverStop)
	kubeInformerFactory.WaitForCacheSync(wait.NeverStop)

	if _, err = kubeInformerFactory.Core().V1().Services().Lister().Services(lbNamespace).Get(lbName); err != nil {
		glog.Fatalf("failed to get service %s/%s from lister: %v", lbNamespace, lbName, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	var gr run.Group
	{
		sig := make(chan os.Signal, 2)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

		gr.Add(func() error {
			<-sig
			return nil
		}, func(err error) {
			cancel()
			close(sig)
		})
	}
	{
		gr.Add(func() error {
			u.Run(1, ctx.Done())
			return nil
		}, func(err error) {
			cancel()
		})
	}

	if err := gr.Run(); err != nil {
		glog.Fatal(err)
	}
}

// LBUpdater has all APIs to synchronize and updateLB the services and nodeports.
type LBUpdater struct {
	queue  workqueue.RateLimitingInterface
	client kubernetes.Interface
	lister corev1lister.ServiceLister

	lbNamespace string
	lbName      string
}

func (u *LBUpdater) enqueue(s *corev1.Service) {
	u.enqueueAfter(s, 0)
}

func (u *LBUpdater) enqueueAfter(s *corev1.Service, duration time.Duration) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(s)
	if err != nil {
		runtime.HandleError(fmt.Errorf("couldn't get key for object %#v: %v", s, err))
		return
	}

	u.queue.AddAfter(key, duration)
}

func (u *LBUpdater) runWorker() {
	for u.processNextItem() {
	}
}

func (u *LBUpdater) processNextItem() bool {
	key, quit := u.queue.Get()
	if quit {
		return false
	}

	defer u.queue.Done(key)

	err := u.syncLB(key.(string))

	u.handleErr(err, key)
	return true
}

// handleErr checks if an error happened and makes sure we will retry later.
func (u *LBUpdater) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		u.queue.Forget(key)
		return
	}

	glog.Errorf("Error syncing Service %v: %v", key, err)

	// Re-enqueue the key rate limited. Based on the rate limiter on the
	// queue and the re-enqueue history, the key will be processed later again.
	u.queue.AddRateLimited(key)
}

// Run starts the controller's worker routines. This method is blocking and ends when stopCh gets closed
func (u *LBUpdater) Run(workerCount int, stopCh <-chan struct{}) {
	defer runtime.HandleCrash()

	for i := 0; i < workerCount; i++ {
		go wait.Until(u.runWorker, time.Second, stopCh)
	}

	<-stopCh
}

func (u *LBUpdater) syncLB(s string) error {
	glog.V(4).Infof("Syncing LB as Service %s got modified", s)

	cacheServices, err := u.lister.List(labels.Everything())
	if err != nil {
		return fmt.Errorf("failed to retrieve services from lister: %v", err)
	}
	services := make([]*corev1.Service, len(cacheServices))
	for i := range cacheServices {
		services[i] = cacheServices[i].DeepCopy()
	}

	var wantLBPorts []corev1.ServicePort
	for _, service := range services {
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

	cacheLb, err := u.lister.Services(u.lbNamespace).Get(u.lbName)
	if err != nil {
		return fmt.Errorf("failed to get service %s/%s from lister: %v", u.lbNamespace, u.lbName, err)
	}
	lb := cacheLb.DeepCopy()

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
		lb, err = u.client.CoreV1().Services(u.lbNamespace).Update(lb)
		if err != nil {
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
