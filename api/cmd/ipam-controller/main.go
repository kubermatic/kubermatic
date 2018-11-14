package main

import (
	"bytes"
	"flag"
	"fmt"
	"net"
	"strings"

	"github.com/kubermatic/kubermatic/api/pkg/util/informer"

	"github.com/golang/glog"

	clusterv1alpha1clientset "github.com/kubermatic/kubermatic/api/pkg/client/cluster-api/clientset/versioned"
	clusterv1alpha1informers "github.com/kubermatic/kubermatic/api/pkg/client/cluster-api/informers/externalversions"
	"github.com/kubermatic/kubermatic/api/pkg/controller/ipam"
	"github.com/kubermatic/kubermatic/api/pkg/leaderelection"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kubeleaderelection "k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/record"

	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
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
		return fmt.Errorf("expected cidr,gateway,dns1,dns2,... but got: %s", value)
	}

	cidrStr := splitted[0]
	ip, ipnet, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return fmt.Errorf("error parsing cidr %s: %v", cidrStr, err)
	}

	gwStr := splitted[1]
	gwIP := net.ParseIP(gwStr)
	if gwIP == nil {
		return fmt.Errorf("expected valid gateway ip but got %s", gwStr)
	}

	dnsSplitted := splitted[2:]
	dnsServers := make([]net.IP, len(dnsSplitted))
	for i, d := range dnsSplitted {
		dnsIP := net.ParseIP(d)
		if dnsIP == nil {
			return fmt.Errorf("expected valid dns ip but got %s", d)
		}

		dnsServers[i] = dnsIP
	}

	val := ipam.Network{
		IP:         ip,
		IPNet:      *ipnet,
		Gateway:    gwIP,
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
	flag.Var(&networks, "network", "The networks from which ips should be allocated (e.g.: ./ipam-controller --network  \"10.0.0.0/16,10.0.0.1,8.8.8.8\" --network  \"192.168.5.0/24,192.168.5.1,1.1.1.1,8.8.4.4\")")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatalf("Couldnt build kubernetes client: %v", err)
	}

	config = restclient.AddUserAgent(config, controllerName)
	client := clusterv1alpha1clientset.NewForConfigOrDie(config)

	err = leaderElectionLoop(config, func(stopCh <-chan struct{}) {
		tweakFunc := func(options *metav1.ListOptions) {
			options.IncludeUninitialized = true
		}

		factory := clusterv1alpha1informers.NewFilteredSharedInformerFactory(client, informer.DefaultInformerResyncPeriod, metav1.NamespaceAll, tweakFunc)
		machineInformer := factory.Cluster().V1alpha1().Machines()

		controller := ipam.NewController(client, machineInformer, networks)

		factory.Start(stopCh)
		factory.WaitForCacheSync(stopCh)
		err := controller.Run(stopCh)
		if err != nil {
			glog.Fatalf("couldn't start controller: %v", err)
		}

		glog.Info("Controller loop finished.")
	})
	if err != nil {
		glog.Fatalf("couldnt start leader election: %v", err)
	}

	glog.Info("Application stopped.")
}

func getEventRecorder(masterKubeClient *kubernetes.Clientset, name string) (record.EventRecorder, error) {
	if err := clusterv1alpha1.AddToScheme(scheme.Scheme); err != nil {
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
