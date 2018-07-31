package ipam

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/golang/glog"

	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"

	machineclientset "github.com/kubermatic/machine-controller/pkg/client/clientset/versioned"
	machineinformersv1alpha1 "github.com/kubermatic/machine-controller/pkg/client/informers/externalversions/machines/v1alpha1"
	machinelistersv1alpha1 "github.com/kubermatic/machine-controller/pkg/client/listers/machines/v1alpha1"
	machinev1alpha1 "github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
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

type Network struct {
	IP         net.IP
	IPNet      *net.IPNet
	Gateway    net.IP
	DNSServers []net.IP
}

type Controller struct {
	queue     workqueue.RateLimitingInterface
	cidrRange []Network

	client        machineclientset.Interface
	machineLister machinelistersv1alpha1.MachineLister

	usedIps map[string]struct{}
}

func NewController(client machineclientset.Interface, machineInformer machineinformersv1alpha1.MachineInformer, networks []Network) *Controller {
	controller := &Controller{
		cidrRange:     networks,
		client:        client,
		machineLister: machineInformer.Lister(),
		usedIps:       make(map[string]struct{}),
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

func (c *Controller) Run(stopCh <-chan struct{}) error {
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
		return c.handleMachineDeletionIfNeeded(m)
	}

	didInitialize, err := c.initMachineIfNeeded(m)
	if err != nil {
		return err
	}

	if !didInitialize {
		err := c.syncIPAllocationFromMachine(m)
		if err != nil {
			return err
		}
	}

	return nil
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

func (c *Controller) handleMachineDeletionIfNeeded(m *machinev1alpha1.Machine) error {
	if !kuberneteshelper.HasFinalizer(m, finalizerName) {
		return nil
	}

	cfg, err := providerconfig.GetConfig(m.Spec.ProviderConfig)
	if err != nil {
		return err
	}

	m.Finalizers = kuberneteshelper.RemoveFinalizer(m.Finalizers, finalizerName)
	_, err = c.client.MachineV1alpha1().Machines().Update(m)
	if err != nil {
		return fmt.Errorf("couldn't update machine %s, see: %v", m.Name, err)
	}

	ip := net.ParseIP(cfg.Network.CIDR)
	c.releaseIP(ip)
	glog.V(6).Infof("Released ip %v from machine %s", ip, m.Name)

	return nil
}

func (c *Controller) writeErrorToMachine(m *machinev1alpha1.Machine, reason machinev1alpha1.MachineStatusError, errToWrite error) error {
	message := errToWrite.Error()
	m.Status.ErrorMessage = &message
	m.Status.ErrorReason = &reason

	_, err := c.client.MachineV1alpha1().Machines().Update(m)
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) syncIPAllocationFromMachine(m *machinev1alpha1.Machine) error {
	cfg, err := providerconfig.GetConfig(m.Spec.ProviderConfig)
	if err != nil {
		return fmt.Errorf("couldn't get provider config: %v", err)
	}

	if cfg.Network == nil {
		return nil
	}

	ip := net.ParseIP(cfg.Network.CIDR)
	if ip == nil {
		return nil
	}

	c.allocateIP(ip)
	glog.V(6).Infof("Synchronized ip %v from machine %s", ip, m.Name)

	return nil
}

func (c *Controller) allocateIP(ip net.IP) {
	c.usedIps[ip.String()] = struct{}{}
}

func (c *Controller) releaseIP(ip net.IP) {
	delete(c.usedIps, ip.String())
}

func (c *Controller) initMachineIfNeeded(oldMachine *machinev1alpha1.Machine) (bool, error) {
	if !c.testIfInitIsNeeded(oldMachine) {
		glog.V(6).Infof("Skipping machine %s because no initialization is needed (yet)", oldMachine.Name)
		return false, nil
	}

	machine := oldMachine.DeepCopy()

	cfg, err := providerconfig.GetConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return false, err
	}

	ip, network := c.getNextFreeIP()
	if ip.IsUnspecified() {
		err = fmt.Errorf("couldn't set ip for %s because no more ips can be allocated from the specified cidrs", machine.Name)
		subErr := c.writeErrorToMachine(machine, machinev1alpha1.InsufficientResourcesMachineError, err)
		if subErr != nil {
			glog.Errorf("couldn't update error state for machine %s, see: %v", machine.Name, subErr)
		}

		return false, err
	}

	cfg.Network = &providerconfig.NetworkConfig{
		CIDR:    ip.String(),
		Gateway: network.Gateway.String(),
		DNS: providerconfig.DNSConfig{
			Servers: c.ipsToStrs(network.DNSServers),
		},
	}

	cfgSerialized, err := json.Marshal(cfg)
	if err != nil {
		return false, err
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
		return false, fmt.Errorf("Couldn't update machine %s, see: %v", machine.Name, err)
	}

	// Having getIP and allocateIP separate will save us from blocking ips even tho we run in kerrors and dont use them.
	// As long as we keep this code blocking & synchronous, this is cool-ish.
	c.allocateIP(ip)
	glog.V(6).Infof("Allocated ip %v for machine %s", ip, oldMachine.Name)

	return true, nil
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

func (c *Controller) getNextFreeIP() (net.IP, Network) {
	for _, cidr := range c.cidrRange {
		ip := c.getNextFreeIPForCIDR(cidr)
		if !ip.IsUnspecified() {
			return ip, cidr
		}
	}

	return net.IP{0, 0, 0, 0}, Network{}
}

func (c *Controller) getNextFreeIPForCIDR(network Network) net.IP {
	for ip := network.IP.Mask(network.IPNet.Mask); network.IPNet.Contains(ip); c.inc(ip) {
		if ip[len(ip)-1] == 0 || ip.Equal(network.Gateway) {
			continue
		}

		if _, used := c.usedIps[ip.String()]; !used {
			return ip
		}
	}

	return net.IP{0, 0, 0, 0}
}

func (c *Controller) inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++

		if ip[j] > 0 {
			break
		}
	}
}
