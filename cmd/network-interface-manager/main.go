/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"time"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"

	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
)

const (
	SCOPE_HOST = 254
)

func main() {
	var (
		mode   string
		ifName string
		ifAddr string
	)

	var link *netlink.Dummy
	var err error
	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)

	flag.StringVar(&mode, "mode", "", "Start mode for the process, it can be init or probe")
	flag.StringVar(&ifName, "if", "", "Name of the network interface to be created or managed, eg: envoyagent")
	flag.StringVar(&ifAddr, "addr", "", "Network Interface address")
	flag.Parse()

	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar()

	// 'init' mode is for init containers
	if mode == "init" {
		link, err = createInterface(ifName)
		if err != nil && !errors.Is(err, fs.ErrExist) {
			log.Fatalf("Failed to create link: %v", err)
			return
		}
		if checkIfAddrExists(link, ifAddr) != nil {
			err = addAddressToInterface(link, ifAddr)
			if err != nil {
				log.Fatalf("Failed to add address to link: %v", err)
			}
			return
		}
	}

	// 'probe' mode is for side-car containers.
	if mode == "probe" {
		ticker := time.NewTicker(time.Second * 10).C
		for {
			<-ticker
			link, err = checkInterfaceExists(ifName)
			if err != nil {
				link, _ = createInterface(ifName)
			}

			err = checkIfAddrExists(link, ifAddr)
			if err != nil {
				_ = addAddressToInterface(link, ifAddr)
			}
		}
	}
}

// Creates a dummy link, equalent to "ip link add envoyagent type dummy".
func createInterface(ifName string) (*netlink.Dummy, error) {
	link := &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: ifName}}

	err := netlink.LinkAdd(link)
	if err != nil {
		return link, fmt.Errorf("could not add %s: %w", link.Name, err)
	}
	log.Printf("Interface %s created", ifName)

	return link, nil
}

// Adds address to the link, equalent to "ip addr add ...".
func addAddressToInterface(link *netlink.Dummy, ifAddr string) error {
	addr := &netlink.Addr{IPNet: &net.IPNet{
		IP:   net.ParseIP(ifAddr),
		Mask: net.CIDRMask(32, 32),
	}}
	addr.Scope = SCOPE_HOST

	// Check for configured addresses and remove them
	addrs, err := netlink.AddrList(link, unix.AF_INET)
	if len(addrs) > 0 {
		for _, val := range addrs {
			err = netlink.AddrDel(link, &val)
			if err != nil {
				return fmt.Errorf("failed to delete address for link: %s err: %w", val, err)
			}
		}
	}

	// Add the requested address to the link
	err = netlink.AddrAdd(link, addr)
	if err != nil {
		return fmt.Errorf("failed to add address to interface %s, error: %w", link.Name, err)
	}
	return err
}

func checkInterfaceExists(ifName string) (*netlink.Dummy, error) {
	link, err := netlink.LinkByName(ifName)
	linkDummy := &netlink.Dummy{LinkAttrs: *link.Attrs()}
	return linkDummy, err
}

func checkIfAddrExists(link *netlink.Dummy, ifAddr string) error {
	addrs, err := netlink.AddrList(link, unix.AF_INET)
	if err != nil {
		return fmt.Errorf("failed to get addresses %s: %w", link.Name, err)
	}

	for _, addr := range addrs {
		err = errors.New("address not present")
		if addr.IPNet.IP.String() == ifAddr {
			err = nil
			break
		}
	}

	return err
}
