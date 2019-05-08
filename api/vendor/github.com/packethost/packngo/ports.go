package packngo

import (
	"fmt"
)

const portBasePath = "/ports"

// DevicePortService handles operations on a port which belongs to a particular device
type DevicePortService interface {
	Assign(*PortAssignRequest) (*Port, *Response, error)
	Unassign(*PortAssignRequest) (*Port, *Response, error)
	Bond(*BondRequest) (*Port, *Response, error)
	Disbond(*DisbondRequest) (*Port, *Response, error)
	DeviceToNetworkType(string, string) (*Device, error)
	DeviceNetworkType(string) (string, error)
	PortToLayerTwo(string) (*Port, *Response, error)
	PortToLayerThree(string) (*Port, *Response, error)
	DeviceToLayerTwo(string) (*Device, error)
	DeviceToLayerThree(string) (*Device, error)
	GetBondPort(string) (*Port, error)
	GetPortByName(string, string) (*Port, error)
}

type PortData struct {
	MAC    string `json:"mac"`
	Bonded bool   `json:"bonded"`
}

type Port struct {
	ID                      string           `json:"id"`
	Type                    string           `json:"type"`
	Name                    string           `json:"name"`
	Data                    PortData         `json:"data"`
	NetworkType             string           `json:"network_type,omitempty"`
	AttachedVirtualNetworks []VirtualNetwork `json:"virtual_networks"`
}

type AddressRequest struct {
	AddressFamily int  `json:"address_family"`
	Public        bool `json:"public"`
}

type BackToL3Request struct {
	RequestIPs []AddressRequest `json:"request_ips"`
}

type DevicePortServiceOp struct {
	client *Client
}

type PortAssignRequest struct {
	PortID           string `json:"id"`
	VirtualNetworkID string `json:"vnid"`
}

type BondRequest struct {
	PortID     string `json:"id"`
	BulkEnable bool   `json:"bulk_enable"`
}

type DisbondRequest struct {
	PortID      string `json:"id"`
	BulkDisable bool   `json:"bulk_disable"`
}

func (i *DevicePortServiceOp) GetBondPort(deviceID string) (*Port, error) {
	device, _, err := i.client.Devices.Get(deviceID, nil)
	if err != nil {
		return nil, err
	}
	for _, port := range device.NetworkPorts {
		if port.Type == "NetworkBondPort" {
			return &port, nil
		}
	}

	return nil, fmt.Errorf("No bonded port found in device %s", deviceID)
}

func (i *DevicePortServiceOp) GetPortByName(deviceID, name string) (*Port, error) {
	device, _, err := i.client.Devices.Get(deviceID, nil)
	if err != nil {
		return nil, err
	}
	for _, port := range device.NetworkPorts {
		if port.Name == name {
			return &port, nil
		}
	}

	return nil, fmt.Errorf("Port %s not found in device %s", name, deviceID)
}

func (i *DevicePortServiceOp) Assign(par *PortAssignRequest) (*Port, *Response, error) {
	path := fmt.Sprintf("%s/%s/assign", portBasePath, par.PortID)
	return i.portAction(path, par)
}

func (i *DevicePortServiceOp) Unassign(par *PortAssignRequest) (*Port, *Response, error) {
	path := fmt.Sprintf("%s/%s/unassign", portBasePath, par.PortID)
	return i.portAction(path, par)
}

func (i *DevicePortServiceOp) Bond(br *BondRequest) (*Port, *Response, error) {
	path := fmt.Sprintf("%s/%s/bond", portBasePath, br.PortID)
	return i.portAction(path, br)
}

func (i *DevicePortServiceOp) Disbond(dr *DisbondRequest) (*Port, *Response, error) {
	path := fmt.Sprintf("%s/%s/disbond", portBasePath, dr.PortID)
	return i.portAction(path, dr)
}

func (i *DevicePortServiceOp) portAction(path string, req interface{}) (*Port, *Response, error) {
	port := new(Port)

	resp, err := i.client.DoRequest("POST", path, req, port)
	if err != nil {
		return nil, resp, err
	}

	return port, resp, err
}

func (i *DevicePortServiceOp) PortToLayerTwo(portID string) (*Port, *Response, error) {
	path := fmt.Sprintf("%s/%s/convert/layer-2", portBasePath, portID)
	port := new(Port)

	resp, err := i.client.DoRequest("POST", path, nil, port)
	if err != nil {
		return nil, resp, err
	}

	return port, resp, err
}

func (i *DevicePortServiceOp) PortToLayerThree(portID string) (*Port, *Response, error) {
	path := fmt.Sprintf("%s/%s/convert/layer-3", portBasePath, portID)
	port := new(Port)

	req := BackToL3Request{
		RequestIPs: []AddressRequest{
			AddressRequest{AddressFamily: 4, Public: true},
			AddressRequest{AddressFamily: 4, Public: false},
			AddressRequest{AddressFamily: 6, Public: true},
		},
	}

	resp, err := i.client.DoRequest("POST", path, &req, port)
	if err != nil {
		return nil, resp, err
	}

	return port, resp, err
}

func (i *DevicePortServiceOp) DeviceNetworkType(deviceID string) (string, error) {
	d, _, err := i.client.Devices.Get(deviceID, nil)
	if err != nil {
		return "", err
	}
	return d.NetworkType, nil
}

func (i *DevicePortServiceOp) DeviceToNetworkType(deviceID string, nType string) (*Device, error) {

	d, _, err := i.client.Devices.Get(deviceID, nil)
	if err != nil {
		return nil, err
	}

	curType := d.NetworkType

	if curType == nType {
		return nil, fmt.Errorf("Device already is in state %s", nType)
	}
	bond0ID := ""
	eth1ID := ""
	for _, port := range d.NetworkPorts {
		if port.Name == "bond0" {
			bond0ID = port.ID
		}
		if port.Name == "eth1" {
			eth1ID = port.ID
		}
	}

	if nType == "layer3" {
		if curType == "layer2-individual" || curType == "layer2-bonded" {
			if curType == "layer2-individual" {
				_, _, err := i.client.DevicePorts.Bond(
					&BondRequest{PortID: bond0ID, BulkEnable: false})
				if err != nil {
					return nil, err
				}

			}
			_, _, err := i.client.DevicePorts.PortToLayerThree(bond0ID)
			if err != nil {
				return nil, err
			}
		}
		_, _, err = i.client.DevicePorts.Bond(
			&BondRequest{PortID: bond0ID, BulkEnable: true})
		if err != nil {
			return nil, err
		}
	}
	if nType == "hybrid" {
		if curType == "layer2-individual" || curType == "layer2-bonded" {
			if curType == "layer2-individual" {
				_, _, err = i.client.DevicePorts.Bond(
					&BondRequest{PortID: bond0ID, BulkEnable: false})
				if err != nil {
					return nil, err
				}
			}
			_, _, err = i.client.DevicePorts.PortToLayerThree(bond0ID)
			if err != nil {
				return nil, err
			}
		}
		_, _, err := i.client.DevicePorts.Disbond(
			&DisbondRequest{PortID: eth1ID, BulkDisable: false})
		if err != nil {
			return nil, err
		}
	}
	if nType == "layer2-individual" {
		if curType == "hybrid" || curType == "layer3" {
			_, _, err = i.client.DevicePorts.PortToLayerTwo(bond0ID)
			if err != nil {
				return nil, err
			}

		}
		_, _, err = i.client.DevicePorts.Disbond(
			&DisbondRequest{PortID: bond0ID, BulkDisable: true})
		if err != nil {
			return nil, err
		}
	}
	if nType == "layer2-bonded" {
		if curType == "hybrid" || curType == "layer3" {
			_, _, err = i.client.DevicePorts.PortToLayerTwo(bond0ID)
			if err != nil {
				return nil, err
			}
		}
		_, _, err = i.client.DevicePorts.Bond(
			&BondRequest{PortID: bond0ID, BulkEnable: false})
		if err != nil {
			return nil, err
		}
	}

	d, _, err = i.client.Devices.Get(deviceID, nil)
	if err != nil {
		return nil, err
	}

	if d.NetworkType != nType {
		return nil, fmt.Errorf(
			"Failed to convert device %s from %s to %s. New type was %s",
			deviceID, curType, nType, d.NetworkType)

	}
	return d, err
}

func (i *DevicePortServiceOp) DeviceToLayerThree(deviceID string) (*Device, error) {
	// hopefull all the VLANs are unassigned at this point
	bond0, err := i.client.DevicePorts.GetBondPort(deviceID)
	if err != nil {
		return nil, err
	}

	bond0, _, err = i.client.DevicePorts.PortToLayerThree(bond0.ID)
	if err != nil {
		return nil, err
	}
	d, _, err := i.client.Devices.Get(deviceID, nil)
	return d, err
}

// DeviceToLayerTwo converts device to L2 networking. Use bond0 to attach VLAN.
func (i *DevicePortServiceOp) DeviceToLayerTwo(deviceID string) (*Device, error) {
	bond0, err := i.client.DevicePorts.GetBondPort(deviceID)
	if err != nil {
		return nil, err
	}

	bond0, _, err = i.client.DevicePorts.PortToLayerTwo(bond0.ID)
	if err != nil {
		return nil, err
	}
	d, _, err := i.client.Devices.Get(deviceID, nil)
	return d, err

}
