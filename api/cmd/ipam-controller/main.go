package main

import (
	"flag"
	"net"
	"strings"
	"time"

	"github.com/golang/glog"

	machineclientset "github.com/kubermatic/machine-controller/pkg/client/clientset/versioned"
	machinev1alpha1 "github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
	//machineinformers "github.com/kubermatic/machine-controller/pkg/client/informers/externalversions"
	//machinelistersv1alpha1 "github.com/kubermatic/machine-controller/pkg/client/listers/machines/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/signals"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	//"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	initializerName = "ipam-controller"
)

var (
	kubeconfig string
	masterURL  string

	cidrRange []net.IPNet
)

func main() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")

	var cidrRangeStr string
	flag.StringVar(&cidrRangeStr, "cidr-range", "", "The range of cidrs from which ips should be dispensed.")

	flag.Parse()

	if cidrRangeStr == "" {
		glog.Fatal("No --cidr-range specified, aborting.")
	}

	cidrs, err := parseCIDRs(cidrRangeStr)
	if err != nil {
		glog.Fatal("Couldn't parse --cidr-range: %v", err)
	}

	cidrRange = cidrs

	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatal("Couldnt build kubernetes client: %v", err)
	}

	machineClient := machineclientset.NewForConfigOrDie(config)
	stopCh := signals.SetupSignalHandler()

	restClient := machineClient.RESTClient()
	watchlist := cache.NewListWatchFromClient(restClient, "machines", corev1.NamespaceAll, fields.Everything())

	// Wrap the returned watchlist to workaround the inability to include
	// the `IncludeUninitialized` list option when setting up watch clients.
	includeUninitializedWatchlist := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.IncludeUninitialized = true
			return watchlist.List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.IncludeUninitialized = true
			return watchlist.Watch(options)
		},
	}

	_, controller := cache.NewInformer(includeUninitializedWatchlist, &machinev1alpha1.Machine{}, 30*time.Second,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				m, ok := obj.(*machinev1alpha1.Machine)
				if !ok {
					glog.Error("got wrong resource in addFunc")
				}

				err := machineAdded(m)
				if err != nil {
					glog.Errorf("error in machineAdded: %v", err)
				}
			},
			DeleteFunc: func(obj interface{}) {
				m, ok := obj.(*machinev1alpha1.Machine)
				if !ok {
					glog.Error("got wrong resource in deleteFunc")
				}

				err := machineDeleted(m)
				if err != nil {
					glog.Errorf("error in machineDeleted: %v", err)
				}
			},
			UpdateFunc: func(oldObj interface{}, newObj interface{}) {
				mOld, ok := oldObj.(*machinev1alpha1.Machine)
				if !ok {
					glog.Error("got wrong resource as oldObj in updateFunc")
				}

				mNew, ok := newObj.(*machinev1alpha1.Machine)
				if !ok {
					glog.Error("got wrong resource as newObj in updateFunc")
				}

				err := machineUpdated(mOld, mNew)
				if err != nil {
					glog.Errorf("error in machineUpdated: %v", err)
				}
			},
		},
	)

	go controller.Run(stopCh)

	<-stopCh
	glog.Info("Application stopped.")
}

func machineAdded(m *machinev1alpha1.Machine) error {
	if !testIfInitIsNeeded(m) {
		return nil
	}

	return nil
}

func machineDeleted(m *machinev1alpha1.Machine) error {
	return nil
}

func machineUpdated(oldM *machinev1alpha1.Machine, newM *machinev1alpha1.Machine) error {
	if !testIfInitIsNeeded(m) {
		return nil
	}

	return nil
}

func testIfInitIsNeeded(m *machinev1alpha1.Machine) bool {
	if m.ObjectMeta.GetInitializers() == nil {
		return false
	}

	pending := m.ObjectMeta.GetInitializers().Pending
	if pending[0].Name != initializerName {
		return false
	}

	return true
}

func parseCIDRs(s string) ([]net.IPNet, error) {
	var cidrs []net.IPNet

	for _, cidrStr := range strings.Split(s, ",") {
		_, ipnet, err := net.ParseCIDR(cidrStr)
		if err != nil {
			return nil, err
		}

		cidrs = append(cidrs, *ipnet)
	}

	return cidrs, nil
}
