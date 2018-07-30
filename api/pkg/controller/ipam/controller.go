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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	initializerName = "ipam.kubermatic.io"
	finalizerName   = initializerName
)

type Network struct {
	ip         net.IP
	ipnet      *net.IPNet
	gateway    net.IP
	dnsServers []net.IP
}

type Controller struct {
	queue     workqueue.RateLimitingInterface
	cidrRange []Network

	client         machineclientset.Interface
	informer       cache.SharedIndexInformer
	lister         machinelistersv1alpha1.MachineLister
	informerSynced cache.InformerSynced

	usedIps map[string]struct{}
}

func NewController(client machineclientset.Interface, informer machineinformersv1alpha1.MachineInformer, networks []Network) *Controller {
	controller := &Controller{
		cidrRange:      networks,
		client:         client,
		informer:       informer.Informer(),
		informerSynced: informer.Informer().Synced,
		lister:         informer.Lister(),
		usedIps:        make(map[string]struct{}),
		queue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "MachineQueue"),
	}

	controller.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.enqueueMachine(obj.(*machinev1alpha1.Machine))
		},
		UpdateFunc: func(_, cur interface{}) {
			c.enqueueMachine(cur.(*machinev1alpha1.Machine))
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

			c.enqueueMachine(m)
		},
	})

	return controller
}

func (c *Controller) Run(stopCh <-chan struct{}) error {
	glog.Info("Starting IPAM-Controller")
	defer glog.Info("Shutting down IPAM-Controller")

	if !cache.WaitForCacheSync(stopCh, c.informerSynced) {
		return errors.New("Unable to sync caches for IPAM-Controller")
	}

	go wait.Until(c.loopQueue, time.Second, stopCh)

	glog.Info("IPAM-Controller started")
	<-stopCh
}

func (c *Controller) loopQueue() {
	stop := false
	for !stop {
		func() {
			key, quit := c.queue.Get()
			if quit {
				stop = true
				return
			}

			defer c.queue.Done(key)

			keyStr := key.(string)
			err := c.handleMachineForKey(keyStr)
			if err != nil {
				glog.Errorf("error syncing machine for key %s: %v", keyStr, err)

				if c.queue.NumRequeues(key) < 5 {
					c.queue.AddRateLimited(key)
					return
				}
			}

			c.queue.Forget(key)
		}()
	}
}

func (c *Controller) handleMachineForKey(key string) {
	m, err := c.informer.Lister().Get(key)

	if err != nil {
		if kerrors.IsNotFound(err) {
			glog.V(2).Infof("machine '%s' no longer exists in queue", key)
			return nil
		}
		return err
	}

	return c.handleMachine(m)
}

func (c *Controller) handleMachine(m *machinev1alpha1.Machine) error {
	if m.DeletionTimestamp != nil {
		err := c.handleMachineDeletionIfNeeded(m)
		return err
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

func (c *Controller) enqueueMachine(m *machinev1alpha1) error {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(m)
	if err != nil {
		return fmt.Errorf("couldn't get key for machine %s: %v", m.Name, err)
	}

	c.queue.Add(key)
}

func (c *Controller) handleMachineDeletionIfNeeded(m *machinev1alpha1) error {
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
		return fmt.Errorf("Couldn't update machine %s, see: %v", m.Name, err)
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
		return fmt.Errorf("Couldn't get provider config: %v", err)
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

	newMachine := oldMachine.DeepCopy()

	cfg, err := providerconfig.GetConfig(newMachine.Spec.ProviderConfig)
	if err != nil {
		return false, err
	}

	ip, network := c.getNextFreeIP()
	if ip.IsUnspecified() {
		err = fmt.Errorf("couldn't set ip for %s because no more ips can be allocated from the specified cidrs", newMachine.Name)
		subErr := c.writeErrorToMachine(client, newMachine, machinev1alpha1.InsufficientResourcesMachineError, err)
		if subErr != nil {
			glog.Errorf("couldn't update error state for machine %s, see: %v", newMachine.Name, subErr)
		}

		return false, err
	}

	cfg.Network = &providerconfig.NetworkConfig{
		CIDR:    ip.String(),
		Gateway: network.gateway.String(),
		DNS: providerconfig.DNSConfig{
			Servers: ipsToStrs(network.dnsServers),
		},
	}

	cfgSerialized, err := json.Marshal(cfg)
	if err != nil {
		return false, err
	}

	newMachine.Finalizers = append(newMachine.Finalizers, finalizerName)
	newMachine.Spec.ProviderConfig = runtime.RawExtension{Raw: cfgSerialized}
	pendingInitializers := newMachine.ObjectMeta.GetInitializers().Pending

	// Remove self from the list of pending Initializers while preserving ordering.
	if len(pendingInitializers) == 1 {
		newMachine.ObjectMeta.Initializers = nil
	} else {
		newMachine.ObjectMeta.Initializers.Pending = append(pendingInitializers[:0], pendingInitializers[1:]...)
	}

	_, err = c.client.MachineV1alpha1().Machines().Update(newMachine)
	if err != nil {
		return false, fmt.Errorf("Couldn't update machine %s, see: %v", newMachine.Name, err)
	}

	// Having getIP and allocateIP separate will save us from blocking ips even tho we run in errors and dont use them.
	// As long as we keep this code blocking & synchronous, this is cool-ish.
	c.allocateIP(ip)
	glog.V(6).Infof("Allocated ip %v for machine %s", ip, oldMachine.Name)

	return true, nil
}

func (c *Controller) ipsToStrs(ips []net.IP) []string {
	strs := make([]string, 0, len(ips))

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

func (c *Controller) getNextFreeIPForCIDR(cidr Network) net.IP {
	for ip := cidr.ip.Mask(cidr.ipnet.Mask); cidr.ipnet.Contains(ip); c.inc(ip) {
		if ip[len(ip)-1] == 0 || ip.Equal(cidr.gateway) {
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
