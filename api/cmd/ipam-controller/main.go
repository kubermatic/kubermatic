package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/golang/glog"

	"github.com/kubermatic/kubermatic/api/pkg/leaderelection"
	machineclientset "github.com/kubermatic/machine-controller/pkg/client/clientset/versioned"
	machinev1alpha1 "github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	kubeleaderelection "k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/record"
)

const (
	initializerName = "ipam.kubermatic.io"
)

type network struct {
	ip    net.IP
	ipnet *net.IPNet
}

var (
	kubeconfig string
	masterURL  string
	gateway    string
	gatewayIP  net.IP
	dnsServers string
	cidrRange  []network

	usedIps map[string]struct{}
)

func main() {
	usedIps = make(map[string]struct{})

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&gateway, "gateway", "", "Ip of the gateway which should be used.")
	flag.StringVar(&dnsServers, "dns-servers", "", "Addresses of the dns servers.")

	var cidrRangeStr string
	flag.StringVar(&cidrRangeStr, "cidr-range", "", "The range of cidrs from which ips should be dispensed.")

	flag.Parse()
	gatewayIP = net.ParseIP(gateway)

	if cidrRangeStr == "" {
		glog.Fatal("No --cidr-range specified, aborting.")
	}

	cidrs, err := parseCIDRs(cidrRangeStr)
	if err != nil {
		glog.Fatalf("Couldn't parse --cidr-range: %v", err)
	}

	cidrRange = cidrs

	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatalf("Couldnt build kubernetes client: %v", err)
	}

	err = leaderElectionLoop(config, func(stop <-chan struct{}) {
		controllerLoop(machineclientset.NewForConfigOrDie(config), stop)
		glog.Info("Controller loop finished.")
	})
	if err != nil {
		glog.Fatalf("couldnt start leader election: %v", err)
	}

	glog.Info("Application stopped.")
}

func controllerLoop(machineClient machineclientset.Interface, stopCh <-chan struct{}) {
	// Wrap the returned watchlist to workaround the inability to include
	// the `IncludeUninitialized` list option when setting up watch clients.
	includeUninitializedWatchlist := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			glog.V(6).Info("Executing List")
			options.IncludeUninitialized = true
			return machineClient.MachineV1alpha1().Machines().List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			glog.V(6).Info("Executing Watch")
			options.IncludeUninitialized = true
			return machineClient.MachineV1alpha1().Machines().Watch(options)
		},
	}

	_, controller := cache.NewInformer(includeUninitializedWatchlist, &machinev1alpha1.Machine{}, 30*time.Second,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				m, ok := obj.(*machinev1alpha1.Machine)
				if !ok {
					glog.Error("got wrong resource in addFunc")
				}

				err := machineAdded(m, machineClient)
				if err != nil {
					glog.Errorf("error in machineAdded: %v", err)
				}
			},
			DeleteFunc: func(obj interface{}) {
				m, ok := obj.(*machinev1alpha1.Machine)
				if !ok {
					glog.Error("got wrong resource in deleteFunc")
				}

				err := machineDeleted(m, machineClient)
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

				err := machineUpdated(mOld, mNew, machineClient)
				if err != nil {
					glog.Errorf("error in machineUpdated: %v", err)
				}
			},
		},
	)

	glog.V(6).Info("Starting controller")
	controller.Run(stopCh)
}

func getEventRecorder(masterKubeClient *kubernetes.Clientset, name string) (record.EventRecorder, error) {
	if err := machinev1alpha1.AddToScheme(scheme.Scheme); err != nil {
		return nil, err
	}
	glog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.V(4).Infof)
	eventBroadcaster.StartRecordingToSink(&corev1.EventSinkImpl{Interface: masterKubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: name})
	return recorder, nil
}

func leaderElectionLoop(config *restclient.Config, callback func(stopCh <-chan struct{})) error {
	name := "ipam-controller"
	leaderElectionClient, err := kubernetes.NewForConfig(restclient.AddUserAgent(config, name))
	if err != nil {
		return err
	}

	callbacks := kubeleaderelection.LeaderCallbacks{
		OnStartedLeading: func(stop <-chan struct{}) {
			callback(stop)
		},
		OnStoppedLeading: func() {
			glog.Error("==================== OnStoppedLeading ====================")
		},
	}

	recorder, err := getEventRecorder(leaderElectionClient, name)
	if err != nil {
		return err
	}

	leader, err := leaderelection.New(name, leaderElectionClient, recorder, callbacks)
	if err != nil {
		return fmt.Errorf("failed to create a leaderelection: %v", err)
	}

	leader.Run()
	return nil
}

func machineAdded(m *machinev1alpha1.Machine, client machineclientset.Interface) error {
	glog.V(6).Infof("machine added %s", m.Name)

	didInitialize, err := initMachineIfNeeded(m, client)
	if err != nil {
		return err
	}

	if !didInitialize {
		err := syncIPAllocationFromMachine(m)
		if err != nil {
			return err
		}
	}

	return nil
}

func machineDeleted(m *machinev1alpha1.Machine, client machineclientset.Interface) error {
	glog.V(6).Infof("machine deleted %s", m.Name)

	cfg, err := providerconfig.GetConfig(m.Spec.ProviderConfig)
	if err != nil {
		return err
	}

	ip := net.ParseIP(cfg.Network.CIDR)

	releaseIP(ip)
	glog.V(6).Infof("Released ip %v from machine %s", ip, m.Name)

	return nil
}

func machineUpdated(oldM *machinev1alpha1.Machine, newM *machinev1alpha1.Machine, client machineclientset.Interface) error {
	glog.V(6).Infof("machine updated %s", newM.Name)

	didInitialize, err := initMachineIfNeeded(newM, client)
	if err != nil {
		return err
	}

	if !didInitialize {
		// no need to sync it, when initMachineIfNeeded already allocated ips.
		err := syncIPAllocationFromMachine(newM)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeErrorToMachine(client machineclientset.Interface, m *machinev1alpha1.Machine, reason machinev1alpha1.MachineStatusError, errToWrite error) error {
	message := errToWrite.Error()
	m.Status.ErrorMessage = &message
	m.Status.ErrorReason = &reason

	_, err := client.MachineV1alpha1().Machines().Update(m)
	if err != nil {
		return err
	}

	return nil
}

func syncIPAllocationFromMachine(m *machinev1alpha1.Machine) error {
	cfg, err := providerconfig.GetConfig(m.Spec.ProviderConfig)
	if err != nil {
		return fmt.Errorf("Couldn't get provider config: %v", err)
	}

	if cfg.Network == nil {
		return nil
	}

	ip := net.ParseIP(cfg.Network.CIDR)
	if ip == nil {
		return nil
	}

	allocateIP(ip)
	glog.V(6).Infof("Synchronized ip %v from machine %s", ip, m.Name)

	return nil
}

func allocateIP(ip net.IP) {
	usedIps[ip.String()] = struct{}{}
}

func releaseIP(ip net.IP) {
	delete(usedIps, ip.String())
}

func initMachineIfNeeded(oldMachine *machinev1alpha1.Machine, client machineclientset.Interface) (bool, error) {
	if !testIfInitIsNeeded(oldMachine) {
		glog.V(6).Infof("Skipping machine %s because no initialization is needed (yet)", oldMachine.Name)
		return false, nil
	}

	newMachine := oldMachine.DeepCopy()

	cfg, err := providerconfig.GetConfig(newMachine.Spec.ProviderConfig)
	if err != nil {
		return false, err
	}

	ip := getNextFreeIP()
	if ip.IsUnspecified() {
		err = fmt.Errorf("couldn't set ip for %s because no more ips can be allocated from the specified cidrs", newMachine.Name)
		subErr := writeErrorToMachine(client, newMachine, machinev1alpha1.InsufficientResourcesMachineError, err)
		if subErr != nil {
			glog.Errorf("couldn't update error state for machine %s, see: %v", newMachine.Name, subErr)
		}

		return false, err
	}

	cfg.Network = &providerconfig.NetworkConfig{
		CIDR:    ip.String(),
		Gateway: gateway,
		DNS: providerconfig.DNSConfig{
			Servers: strings.Split(dnsServers, ","),
		},
	}

	cfgSerialized, err := json.Marshal(cfg)
	if err != nil {
		return false, err
	}

	newMachine.Spec.ProviderConfig = runtime.RawExtension{Raw: cfgSerialized}
	pendingInitializers := newMachine.ObjectMeta.GetInitializers().Pending

	// Remove self from the list of pending Initializers while preserving ordering.
	if len(pendingInitializers) == 1 {
		newMachine.ObjectMeta.Initializers = nil
	} else {
		newMachine.ObjectMeta.Initializers.Pending = append(pendingInitializers[:0], pendingInitializers[1:]...)
	}

	_, err = client.MachineV1alpha1().Machines().Update(newMachine)
	if err != nil {
		return false, fmt.Errorf("Couldn't update machine %s, see: %v", newMachine.Name, err)
	}

	// Having getIP and allocateIP separate will save us from blocking ips even tho we run in errors and dont use them.
	// As long as we keep this code blocking & synchronous, this is cool-ish.
	allocateIP(ip)
	glog.V(6).Infof("Allocated ip %v for machine %s", ip, oldMachine.Name)

	return true, nil
}

func testIfInitIsNeeded(m *machinev1alpha1.Machine) bool {
	if m.ObjectMeta.GetInitializers() == nil {
		return false
	}

	return m.ObjectMeta.GetInitializers().Pending[0].Name == initializerName
}

func getNextFreeIP() net.IP {
	for _, cidr := range cidrRange {
		ip := getNextFreeIPForCIDR(cidr)
		if !ip.IsUnspecified() {
			return ip
		}
	}

	return net.IP{0, 0, 0, 0}
}

func getNextFreeIPForCIDR(cidr network) net.IP {
	for ip := cidr.ip.Mask(cidr.ipnet.Mask); cidr.ipnet.Contains(ip); inc(ip) {
		if ip[len(ip)-1] == 0 || ip.Equal(gatewayIP) {
			continue
		}

		if _, used := usedIps[ip.String()]; !used {
			return ip
		}
	}

	return net.IP{0, 0, 0, 0}
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++

		if ip[j] > 0 {
			break
		}
	}
}

func parseCIDRs(s string) ([]network, error) {
	var cidrs []network

	for _, cidrStr := range strings.Split(s, ",") {
		ip, ipnet, err := net.ParseCIDR(cidrStr)
		if err != nil {
			return nil, err
		}

		cidrs = append(cidrs, network{ip: ip, ipnet: ipnet})
	}

	return cidrs, nil
}
