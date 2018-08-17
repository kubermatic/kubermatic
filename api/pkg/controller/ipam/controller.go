package ipam

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/golang/glog"

	machineclientset "github.com/kubermatic/machine-controller/pkg/client/clientset/versioned"
	machineinformersv1alpha1 "github.com/kubermatic/machine-controller/pkg/client/informers/externalversions/machines/v1alpha1"
	machinelistersv1alpha1 "github.com/kubermatic/machine-controller/pkg/client/listers/machines/v1alpha1"
	machinev1alpha1 "github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	initializerName = "ipam.kubermatic.io"
	finalizerName   = initializerName
)

type cidrExhaustedError struct{}

func (c cidrExhaustedError) Error() string {
	return "cidr exhausted"
}

// Network represents a machine network configuration
type Network struct {
	IP         net.IP
	IPNet      net.IPNet
	Gateway    net.IP
	DNSServers []net.IP
}

// Controller is the ipam controller itself
type Controller struct {
	queue     workqueue.RateLimitingInterface
	cidrRange []Network

	client        machineclientset.Interface
	machineLister machinelistersv1alpha1.MachineLister
}

// NewController creates a new controller for the specified data.
func NewController(client machineclientset.Interface, machineInformer machineinformersv1alpha1.MachineInformer, networks []Network) *Controller {
	controller := &Controller{
		cidrRange:     networks,
		client:        client,
		machineLister: machineInformer.Lister(),
		queue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "MachineQueue"),
	}

	machineInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			controller.enqueueMachine(obj.(*machinev1alpha1.Machine))
		},
		UpdateFunc: func(_, cur interface{}) {
			controller.enqueueMachine(cur.(*machinev1alpha1.Machine))
		},
		DeleteFunc: func(obj interface{}) {
			m, ok := obj.(*machinev1alpha1.Machine)
			if !ok {
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.Errorf("couldn't get object from tombstone %#v", obj)
					return
				}
				m, ok = tombstone.Obj.(*machinev1alpha1.Machine)
				if !ok {
					glog.Errorf("tombstone contained object that is not a machine %#v", obj)
					return
				}
			}

			controller.enqueueMachine(m)
		},
	})

	return controller
}

// Run executes the worker loop. Blocks.
func (c *Controller) Run(stopCh <-chan struct{}) error {
	// ATM it is important that only one worker is running at a time since we dont do any locking for the "wait till cache has synchronized"-mechanism which would be needed if we have multiple workers running.
	go wait.Until(c.runWorker, time.Second, stopCh)
	<-stopCh

	return nil
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}

	defer c.queue.Done(key)

	glog.V(6).Infof("Processing machine: %s", key)
	err := c.syncHandler(key.(string))
	if err == nil {
		c.queue.Forget(key)
		return true
	}

	utilruntime.HandleError(fmt.Errorf("%v failed with: %v", key, err))
	c.queue.AddRateLimited(key)

	return true
}

func (c *Controller) syncHandler(key string) error {
	listerMachine, err := c.machineLister.Get(key)

	if err != nil {
		if kerrors.IsNotFound(err) {
			glog.V(2).Infof("machine '%s' in work queue no longer exists", key)
			return nil
		}
		return err
	}

	return c.syncMachine(listerMachine)
}

func (c *Controller) syncMachine(mo *machinev1alpha1.Machine) error {
	m := mo.DeepCopy()

	if m.DeletionTimestamp != nil {
		return nil
	}

	return c.initMachineIfNeeded(m)
}

func (c *Controller) enqueueMachine(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.queue.AddRateLimited(key)
}

func (c *Controller) writeErrorToMachine(m *machinev1alpha1.Machine, reason machinev1alpha1.MachineStatusError, errToWrite error) error {
	message := errToWrite.Error()
	m.Status.ErrorMessage = &message
	m.Status.ErrorReason = &reason

	_, err := c.client.MachineV1alpha1().Machines().Update(m)
	return err
}

func (c *Controller) getUsedIPs() ([]net.IP, error) {
	machines, err := c.machineLister.List(labels.Everything())
	if err != nil {
		return nil, fmt.Errorf("error listing machines: '%v'", err)
	}

	ips := make([]net.IP, 0)
	for _, m := range machines {
		if m.DeletionTimestamp != nil {
			continue
		}

		cfg, err := providerconfig.GetConfig(m.Spec.ProviderConfig)
		if err != nil {
			return nil, err
		}

		if cfg.Network == nil {
			continue
		}

		ip, _, err := net.ParseCIDR(cfg.Network.CIDR)
		if err != nil {
			return nil, err
		}

		if ip == nil {
			continue
		}

		ips = append(ips, ip)
	}

	return ips, nil
}

func (c *Controller) initMachineIfNeeded(machine *machinev1alpha1.Machine) error {
	if !c.testIfInitIsNeeded(machine) {
		glog.V(6).Infof("Skipping machine %s because no initialization is needed (yet)", machine.Name)
		return nil
	}

	cfg, err := providerconfig.GetConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return err
	}

	ip, network, err := c.getNextFreeIP()
	if _, isCidrExhausted := err.(cidrExhaustedError); isCidrExhausted {
		err = fmt.Errorf("couldn't set ip for %s because no more ips can be allocated from the specified cidrs", machine.Name)
		subErr := c.writeErrorToMachine(machine, machinev1alpha1.InsufficientResourcesMachineError, err)
		if subErr != nil {
			glog.Errorf("couldn't update error state for machine %s, see: %v", machine.Name, subErr)
		}

		return err
	}

	mask, _ := network.IPNet.Mask.Size()
	cidr := fmt.Sprintf("%s/%d", ip.String(), mask)

	cfg.Network = &providerconfig.NetworkConfig{
		CIDR:    cidr,
		Gateway: network.Gateway.String(),
		DNS: providerconfig.DNSConfig{
			Servers: c.ipsToStrs(network.DNSServers),
		},
	}

	cfgSerialized, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	machine.Finalizers = append(machine.Finalizers, finalizerName)
	machine.Spec.ProviderConfig = runtime.RawExtension{Raw: cfgSerialized}
	pendingInitializers := machine.ObjectMeta.GetInitializers().Pending

	// Remove self from the list of pending Initializers while preserving ordering.
	if len(pendingInitializers) == 1 {
		machine.ObjectMeta.Initializers = nil
	} else {
		machine.ObjectMeta.Initializers.Pending = append(pendingInitializers[:0], pendingInitializers[1:]...)
	}

	_, err = c.client.MachineV1alpha1().Machines().Update(machine)
	if err != nil {
		return fmt.Errorf("Couldn't update machine %s, see: %v", machine.Name, err)
	}

	return c.awaitIPSync(machine, cidr)
}

func (c *Controller) awaitIPSync(machine *machinev1alpha1.Machine, cidr string) error {
	return wait.Poll(10*time.Millisecond, 60*time.Second, func() (bool, error) {
		key, err := cache.MetaNamespaceKeyFunc(machine)
		if err != nil {
			return false, fmt.Errorf("something terrible happened - meta for machine %s got erased", machine.Name)
		}

		m2, err := c.machineLister.Get(key)
		if err != nil {
			return false, fmt.Errorf("error while retrieving machine %s from lister, see: %v", m2.Name, err)
		}

		cfg2, err := providerconfig.GetConfig(m2.Spec.ProviderConfig)
		if err != nil {
			return false, fmt.Errorf("couldn't get providerconfig for machine %s, see: %v", m2.Name, err)
		}

		return cfg2.Network != nil && cfg2.Network.CIDR == cidr, nil
	})
}

func (c *Controller) ipsToStrs(ips []net.IP) []string {
	strs := make([]string, len(ips))

	for i, ip := range ips {
		strs[i] = ip.String()
	}

	return strs
}

func (c *Controller) testIfInitIsNeeded(m *machinev1alpha1.Machine) bool {
	if m.ObjectMeta.GetInitializers() == nil {
		return false
	}

	return m.ObjectMeta.GetInitializers().Pending[0].Name == initializerName
}

func (c *Controller) getNextFreeIP() (net.IP, Network, error) {
	usedIps, err := c.getUsedIPs()
	if err != nil {
		return nil, Network{}, err
	}

	for _, cidr := range c.cidrRange {
		ip, err := c.getNextFreeIPForCIDR(cidr, usedIps)
		if err == nil {
			return ip, cidr, nil
		}
	}

	return nil, Network{}, cidrExhaustedError{}
}

func (c *Controller) getNextFreeIPForCIDR(network Network, usedIps []net.IP) (net.IP, error) {
	for ip := network.IP.Mask(network.IPNet.Mask); network.IPNet.Contains(ip); c.inc(ip) {
		if ip[len(ip)-1] == 0 || ip[len(ip)-1] == 255 || ip.Equal(network.Gateway) {
			continue
		}

		if !ipsContains(usedIps, ip) {
			return ip, nil
		}
	}

	return nil, cidrExhaustedError{}
}

func ipsContains(haystack []net.IP, needle net.IP) bool {
	for _, ip := range haystack {
		if ip.Equal(needle) {
			return true
		}
	}

	return false
}

func (c *Controller) inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++

		if ip[j] > 0 {
			break
		}
	}
}
