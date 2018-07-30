package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"

	"github.com/kubermatic/kubermatic/api/pkg/controller/ipam"
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

const controllerName = "ipam-controller"

func main() {
	networks, kubeconfig, masterURL, err := parseArgs(os.Args[1:])
	if err != nil {
		glog.Errorf("couldn't parse args: %v", err)
		glog.Infoln("usage: ./ipam-controller --cidr \"192.168.0.14/30\" --gateway \"192.168.0.1\" --dns-servers \"192.168.0.1,192.168.0.2\" --cidr \"10.0.0.0/16\" --gateway \"10.0.0.1\" --dns-server \"8.8.8.8,8.8.4.4\"")
		os.Exit(1)
	}

	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatalf("Couldnt build kubernetes client: %v", err)
	}

	config = restclient.AddUserAgent(config, controllerName)
	client := machineclientset.NewForConfigOrDie(config)
	controller := setupController(client, networks)

	err = leaderElectionLoop(config, func(stopCh <-chan struct{}) {
		controller.Run(stopCh)

		glog.Info("Controller loop finished.")
	})
	if err != nil {
		glog.Fatalf("couldnt start leader election: %v", err)
	}

	glog.Info("Application stopped.")
}

func setupController(client machineclientset.Interface, networks []Network) *ipam.Controller {
	tweakFunc := func(options metav1.ListOptions) {
		options.IncludeUninitialized = true
	}

	factory := machinev1alpha1.NewFilteredSharedInformerFactory(client, 30*time.Second(), metav1.NamespaceAll, tweakFunc)
	informer := factory.Machine().V1alpha1().Machines()

	return ipam.New(client, informer, networks)
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
	leaderElectionClient, err := kubernetes.NewForConfig(config)
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

	recorder, err := getEventRecorder(leaderElectionClient, controllerName)
	if err != nil {
		return err
	}

	leader, err := leaderelection.New(controllerName, leaderElectionClient, recorder, callbacks)
	if err != nil {
		return fmt.Errorf("failed to create a leaderelection: %v", err)
	}

	leader.Run()
	return nil
}

func parseArgs(args []string) (list []ipam.Network, kubecfg string, masterUrl string, retErr error) {
	kubecfg = ""
	masterUrl = ""
	retErr = nil
	list = make([]ipam.Network)
	var current *ipam.Network

	getNetErrors := func(n *ipam.Network) error {
		if n.gateway == nil || n.gateway.IsUnspecified() {
			return fmt.Errorf("missing gateway for %v", n.ip)
		}

		if n.dnsServers == nil || len(n.dnsServers) == 0 {
			return fmt.Errorf("missing dns servers for %v", n.ip)
		}
	}

	// manual parsing so we can supply multiple params for the different cidrs.
	for i := 0; i < len(args); i += 2 {
		arg = args[i]

		hasNextArg = len(args) > i+1
		if !hasNextArg {
			retErr = fmt.Errorf("expected value for parameter %s", arg)
			return
		}

		nextArg = args[i+1]

		if arg == "--cidr" {
			if current != nil {
				if err := getNetErrors(current); err != nil {
					retErr = err
					return
				}

				list = append(list, *current)
			}

			ip, ipnet, err := net.ParseCIDR(nextArg)
			if err != nil {
				retErr = fmt.Errorf("error parsing cidr %s: %v", nextArg, err)
				return
			}

			current := &ipam.Network{
				ip:    ip,
				ipnet: ipnet,
			}
		} else if arg == "--gateway" {
			if current == nil {
				retErr = errors.New("wrong order, expected --cidr before --gateway")
				return
			}

			current.gateway = net.ParseIP(nextArg)
		} else if arg == "--dns-servers" {
			if current == nil {
				retErr = errors.New("wrong order, expected --cidr before --dns-servers")
				return
			}

			ipStrs := strings.Split(nextArg, ",")
			dnsServers := make([]net.IP, 0, len(ipStrs))
			for i2, v := range ipStrs {
				dnsServers[i2] = net.ParseIP(v)
			}

			current.dnsServers = dnsServers
		} else if arg == "--kubeconfig" {
			kubecfg = nextArg
		} else if arg == "--master" {
			masterUrl = nextArg
		} else {
			retErr = fmt.Errrof("unknown flag: %s", arg)
			return
		}
	}

	if err := getNetErrors(current); err != nil {
		retErr = err
		return
	}

	list = append(list, *current)

	return
}
