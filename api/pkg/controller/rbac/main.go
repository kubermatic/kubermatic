package main

import (
	"flag"
	"github.com/golang/glog"
	"time"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"fmt"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached"
)

const ResourceResyncTime time.Duration = 0

type queueItem struct {
	obj       interface{}
	gvr schema.GroupVersionResource
	kind string
}

type tuple struct {
	gvr schema.GroupVersionResource
	kind string
}

func main() {
	var kubeconfig string
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.Parse()
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}


	// create dynamic client
	client := discovery.NewDiscoveryClientForConfigOrDie(config)
	cacheDiscoClient := cached.NewMemCacheClient(client)
	restMapper := discovery.NewDeferredDiscoveryRESTMapper(cacheDiscoClient, meta.InterfacesForUnstructured)
	restMapper.Reset()
	clientPool := dynamic.NewClientPool(config, restMapper, dynamic.LegacyAPIPathResolverFunc)

	kubermaticClient := kubermaticclientset.NewForConfigOrDie(config)
	kubermaticSharedInformers := externalversions.NewSharedInformerFactory(kubermaticClient, time.Minute*5)
	queue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "rbac_controller_dependant_resources")

	// a list of resources we want to monitor
	// this is essentially a list of project dependant resources
	// like cluster ans ssh keys
	resources := []tuple{}
	resources = append(resources, tuple{gvr:schema.GroupVersionResource{
		Group:"kubermatic.k8s.io",
		Version: "v1",
		Resource: "projects",
	}, kind: "Project"})
	resources = append(resources, tuple{gvr:schema.GroupVersionResource{
		Group:    "kubermatic.k8s.io",
		Version:  "v1",
		Resource: "clusters",
	}, kind: "Cluster"})

	controllers := []cache.Controller{}
	for _, resource := range resources {
		controller, err := controllerFor(queue, kubermaticSharedInformers, resource)
		if err != nil {
			glog.Fatal(err)
		}
		controllers = append(controllers, controller)
	}

	stopCh := make(chan struct{})
	kubermaticSharedInformers.Start(stopCh)
	for _, controller := range controllers {
		controller.Run(stopCh)
	}


	// find better way of waiting for all controllers/informers
	for _, controller := range controllers {
		if !cache.WaitForCacheSync(stopCh, controller.HasSynced) {
			glog.Fatal("Unable to sync caches for RBACGenerator controller")
		}
	}

	for {
		err := process(queue, clientPool)
		if err != nil {
			fmt.Println(err)
		}
	}

}

func process(queue workqueue.RateLimitingInterface, dClientPool dynamic.ClientPool) error {
	qItem, quit := queue.Get()
	if quit {
		return fmt.Errorf("quit requested")
	}
	defer queue.Done(qItem)

	item, ok := qItem.(*queueItem)
	if !ok {
		return fmt.Errorf("item in the queue is not *queueItem")
	}
	accessor, err := meta.Accessor(item.obj)
	if err != nil {
		return fmt.Errorf("cannot access obj: %v", err)
	}
	fmt.Println(fmt.Sprintf("processing object: namespace %s, name %s, uid %s, kind %s, gvr %s",
		accessor.GetNamespace(),
		accessor.GetName(),
		string(accessor.GetUID()),
		item.kind,
		item.gvr.String()))

	// for each item/ resource we will call
	//
	// ensureRBACRoleBinding
	// ensureRBACRole
	//
	// if there is a need to modify the resource here is how we could do it using dynamic client
	return processInternal(item.obj, item.gvr.GroupVersion().WithKind(item.kind), item.gvr, dClientPool)
}

func processInternal(obj interface{}, gvk schema.GroupVersionKind, gvr schema.GroupVersionResource, dClientPool dynamic.ClientPool) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return fmt.Errorf("cannot access obj: %v", err)
	}
	client, err := dClientPool.ClientForGroupVersionKind(gvk)
	if err != nil {
		return err
	}
	resource := &metav1.APIResource{
		Name:       gvr.Resource,
		Namespaced: false,
		Kind:       gvk.Kind,
		Group:      gvk.Group,
		Version:    gvk.Version,
	}
	_, err = client.Resource(resource, accessor.GetNamespace()).Get(accessor.GetName(), metav1.GetOptions{})
	return err
}

func controllerFor(queue workqueue.RateLimitingInterface, sharedInformers externalversions.SharedInformerFactory, resource tuple) (cache.Controller, error) {
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			fmt.Println("adding obje to the queue")
			item := &queueItem{
				gvr: resource.gvr,
				kind: resource.kind,
				obj: obj,
			}
			queue.Add(item)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			fmt.Println("updating obje to the queue")
			item := &queueItem{
				gvr:  resource.gvr,
				kind: resource.kind,
				obj:  newObj,
			}
			queue.Add(item)
		},
		DeleteFunc: func(obj interface{}) {
			if deletedFinalStateUnknown, ok := obj.(cache.DeletedFinalStateUnknown); ok {
				obj = deletedFinalStateUnknown.Obj
			}
			fmt.Println("dlete obje to the queue")
			item := &queueItem{
				gvr:  resource.gvr,
				kind: resource.kind,
				obj:  obj,
			}
			queue.Add(item)
		},
	}
	shared, err := sharedInformers.ForResource(resource.gvr)
	if err == nil {
		glog.V(4).Infof("using a shared informer for resource %q", resource.gvr.String())
		// need to clone because it's from a shared cache
		shared.Informer().AddEventHandlerWithResyncPeriod(handlers, ResourceResyncTime)
		return shared.Informer().GetController(), nil
	}
	return nil, fmt.Errorf("uanble to create shared informer fo the given resourece %v", resource.gvr.String())
}
