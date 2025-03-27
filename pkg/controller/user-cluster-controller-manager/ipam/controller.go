/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ipam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"go.uber.org/zap"

	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	"k8c.io/machine-controller/sdk/providerconfig"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// ControllerName is the name of this controller.
	ControllerName                 = "kkp-ipam-controller"
	annotationMachineUninitialized = "machine-controller.kubermatic.io/initializers"
	annotationValue                = "ipam"
)

// Network represents a machine network configuration.
type Network struct {
	IP         net.IP
	IPNet      net.IPNet
	Gateway    net.IP
	DNSServers []net.IP
}

type reconciler struct {
	ctrlruntimeclient.Client
	recorder   record.EventRecorder
	cidrRanges []Network
	log        *zap.SugaredLogger
}

func Add(mgr manager.Manager, cidrRanges []Network, log *zap.SugaredLogger) error {
	reconciler := &reconciler{Client: mgr.GetClient(),
		recorder:   mgr.GetEventRecorderFor(ControllerName),
		cidrRanges: cidrRanges,
		log:        log,
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		For(&clusterv1alpha1.Machine{}).
		Build(reconciler)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("machine", request)
	log.Debug("Reconciling")

	machine := &clusterv1alpha1.Machine{}
	if err := r.Get(ctx, request.NamespacedName, machine); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	err := r.reconcile(ctx, log, machine)
	if err != nil {
		r.recorder.Event(machine, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, machine *clusterv1alpha1.Machine) error {
	if machine.DeletionTimestamp != nil {
		return nil
	}

	if !strings.Contains(machine.Annotations[annotationMachineUninitialized], annotationValue) {
		log.Debug("Machine doesn't need initialization")
		return nil
	}

	cfg, err := providerconfig.GetConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return err
	}

	ip, network, err := r.getNextFreeIP(ctx)
	if err != nil {
		return err
	}

	mask, _ := network.IPNet.Mask.Size()
	cidr := fmt.Sprintf("%s/%d", ip.String(), mask)

	cfg.Network = &providerconfig.NetworkConfig{
		CIDR:    cidr,
		Gateway: network.Gateway.String(),
		DNS: providerconfig.DNSConfig{
			Servers: r.ipsToStrs(network.DNSServers),
		},
	}

	cfgSerialized, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	machine.Spec.ProviderSpec.Value = &runtime.RawExtension{Raw: cfgSerialized}
	newAnnotationVal := strings.ReplaceAll(machine.Annotations[annotationMachineUninitialized], annotationValue, "")
	machine.Annotations[annotationMachineUninitialized] = newAnnotationVal
	if err := r.Update(ctx, machine); err != nil {
		return fmt.Errorf("failed to update machine after adding network: %w", err)
	}

	// Block until the change is in the lister to make sure we don't hand out an IP twice
	return wait.PollUntilContextTimeout(ctx, 1*time.Second, 60*time.Second, false, func(ctx context.Context) (bool, error) {
		newMachine := &clusterv1alpha1.Machine{}
		if err := r.Get(ctx, types.NamespacedName{Namespace: machine.Namespace, Name: machine.Name}, newMachine); err != nil {
			return false, err
		}
		newConfig, err := providerconfig.GetConfig(newMachine.Spec.ProviderSpec)
		if err != nil {
			return false, nil
		}
		return newConfig.Network != nil && newConfig.Network.CIDR == cidr, nil
	})
}

func (r *reconciler) getNextFreeIP(ctx context.Context) (net.IP, Network, error) {
	usedIps, err := r.getUsedIPs(ctx)
	if err != nil {
		return nil, Network{}, err
	}

	for _, cidr := range r.cidrRanges {
		ip, err := r.getNextFreeIPForCIDR(cidr, usedIps)
		if err == nil {
			return ip, cidr, nil
		}
	}

	return nil, Network{}, errors.New("cidr exhausted")
}

func (r *reconciler) ipsToStrs(ips []net.IP) []string {
	strs := make([]string, len(ips))

	for i, ip := range ips {
		strs[i] = ip.String()
	}

	return strs
}

func (r *reconciler) getUsedIPs(ctx context.Context) ([]net.IP, error) {
	machines := &clusterv1alpha1.MachineList{}
	if err := r.List(ctx, machines); err != nil {
		return nil, fmt.Errorf("failed to list machines: %w", err)
	}

	ips := make([]net.IP, 0)
	for _, m := range machines.Items {
		if m.DeletionTimestamp != nil {
			continue
		}

		cfg, err := providerconfig.GetConfig(m.Spec.ProviderSpec)
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

func (r *reconciler) getNextFreeIPForCIDR(network Network, usedIps []net.IP) (net.IP, error) {
	for ip := network.IP.Mask(network.IPNet.Mask); network.IPNet.Contains(ip); inc(ip) {
		if ip[len(ip)-1] == 0 || ip[len(ip)-1] == 255 || ip.Equal(network.Gateway) {
			continue
		}

		if !ipsContains(usedIps, ip) {
			return ip, nil
		}
	}

	return nil, errors.New("cidr exhausted")
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++

		if ip[j] > 0 {
			break
		}
	}
}

func ipsContains(haystack []net.IP, needle net.IP) bool {
	for _, ip := range haystack {
		if ip.Equal(needle) {
			return true
		}
	}

	return false
}
