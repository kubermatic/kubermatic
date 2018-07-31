package main

import (
	"bytes"
	"flag"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/golang/glog"

	"github.com/kubermatic/kubermatic/api/pkg/controller/ipam"
	"github.com/kubermatic/kubermatic/api/pkg/leaderelection"
	machineclientset "github.com/kubermatic/machine-controller/pkg/client/clientset/versioned"
	machineinformers "github.com/kubermatic/machine-controller/pkg/client/informers/externalversions"
	machinev1alpha1 "github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kubeleaderelection "k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/record"
)

const controllerName = "ipam-controller"

type networkFlags []ipam.Network

func (nf *networkFlags) String() string {
	var buf bytes.Buffer

	for i, n := range *nf {
		buf.WriteString(n.IP.String())
		buf.WriteString(",")
		buf.WriteString(n.Gateway.String())
		buf.WriteString(",")

		for iD, dns := range n.DNSServers {
			buf.WriteString(dns.String())

			if iD < len(n.DNSServers)-1 {
				buf.WriteString(",")
			}
		}

		if i < len(*nf)-1 {
			buf.WriteString(";")
		}
	}

	return buf.String()
}

func (nf *networkFlags) Set(value string) error {
	splitted := strings.Split(value, ",")

	if len(splitted) < 3 {
		return fmt.Errorf("Expected cidr,gateway,dns1,dns2,... but got: %s", value)
	}

	cidrStr := splitted[0]
	ip, ipnet, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return fmt.Errorf("error parsing cidr %s: %v", cidrStr, err)
	}

	gwStr := splitted[1]
	gwIp := net.ParseIP(gwStr)
	if gwIp == nil {
		return fmt.Errorf("expected valid gateway ip but got %s", gwStr)
	}

	dnsSplitted := splitted[2:]
	dnsServers := make([]net.IP, len(dnsSplitted))
	for i, d := range dnsSplitted {
		dnsIp := net.ParseIP(d)
		if dnsIp == nil {
			return fmt.Errorf("expected valid dns ip but got %s", d)
		}

		dnsServers[i] = dnsIp
	}

	val := ipam.Network{
		IP:         ip,
		IPNet:      ipnet,
		Gateway:    gwIp,
		DNSServers: dnsServers,
	}

	*nf = append(*nf, val)
	return nil
}

func main() {
	var networks networkFlags
	var kubeconfig, masterURL string

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.Var(&networks, "network", "The networks from which ips should be allocated (format: cidr,gw,dns1,dns2,...)")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatalf("Couldnt build kubernetes client: %v", err)
	}

	config = restclient.AddUserAgent(config, controllerName)
	client := machineclientset.NewForConfigOrDie(config)

	err = leaderElectionLoop(config, func(stopCh <-chan struct{}) {
		tweakFunc := func(options *metav1.ListOptions) {
			options.IncludeUninitialized = true
		}

		factory := machineinformers.NewFilteredSharedInformerFactory(client, 30*time.Second, metav1.NamespaceAll, tweakFunc)
		informer := factory.Machine().V1alpha1().Machines()

		controller := ipam.NewController(client, informer, networks)

		factory.Start(stopCh)
		controller.Run(stopCh)

		glog.Info("Controller loop finished.")
	})
	if err != nil {
		glog.Fatalf("couldnt start leader election: %v", err)
	}

	glog.Info("Application stopped.")
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
