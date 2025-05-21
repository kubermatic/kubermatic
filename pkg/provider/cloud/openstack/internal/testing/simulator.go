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

package testing

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/subnetpools"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	ostesthelper "github.com/gophercloud/gophercloud/testhelper"
	osfakeclient "github.com/gophercloud/gophercloud/testhelper/client"
)

type Resource interface {
	GetType() string
	GetPath() string
	GetID() string
	GetName() string
	FromCreateRequest([]byte) (Resource, error)
	CreateResponse() ([]byte, error)
	SubResources() []ResourceBuilder
}

type ResourceBuilder func() Resource

type Network struct {
	external.NetworkExternalExt
	networks.Network
}

func (n Network) GetID() string {
	return n.ID
}

func (n Network) GetName() string {
	return n.Name
}

func (n Network) GetType() string {
	return "networks"
}

func (n Network) GetPath() string {
	return "/networks"
}

func (n *Network) FromCreateRequest(c []byte) (Resource, error) {
	createOpts := struct {
		networks.CreateOpts `json:"network"`
	}{}
	err := json.Unmarshal(c, &createOpts)
	if err != nil {
		return nil, err
	}
	n.ID = NetworkID
	n.Name = createOpts.Name
	n.Description = createOpts.Description
	n.ProjectID = createOpts.ProjectID
	n.UpdatedAt = time.Now()
	n.CreatedAt = time.Now()
	return n, nil
}

func (n *Network) CreateResponse() ([]byte, error) {
	res := struct {
		Network *Network `json:"network"`
	}{
		Network: n,
	}
	return json.Marshal(res)
}

func (n *Network) SubResources() []ResourceBuilder {
	return nil
}

type SecGroup groups.SecGroup

func (s SecGroup) GetID() string {
	return s.ID
}

func (s SecGroup) GetName() string {
	return s.Name
}

func (s SecGroup) GetType() string {
	return "security_groups"
}

func (s SecGroup) GetPath() string {
	return "/security-groups"
}

func (s *SecGroup) FromCreateRequest(c []byte) (Resource, error) {
	createOpts := struct {
		groups.CreateOpts `json:"security_group"`
	}{}
	err := json.Unmarshal(c, &createOpts)
	if err != nil {
		return nil, err
	}
	s.ID = SecGroupID
	s.Name = createOpts.Name
	s.Description = createOpts.Description
	s.ProjectID = createOpts.ProjectID
	s.UpdatedAt = time.Now()
	s.CreatedAt = time.Now()
	return s, nil
}

func (s *SecGroup) CreateResponse() ([]byte, error) {
	res := struct {
		SecGroup *SecGroup `json:"security_group"`
	}{
		SecGroup: s,
	}
	return json.Marshal(res)
}

func (s *SecGroup) SubResources() []ResourceBuilder {
	return nil
}

type SecGroupRule rules.SecGroupRule

func (s SecGroupRule) GetID() string {
	return s.ID
}

func (s SecGroupRule) GetName() string {
	return ""
}

func (s SecGroupRule) GetType() string {
	return "security_group_rules"
}

func (s SecGroupRule) GetPath() string {
	return "/security-group-rules"
}

func (s *SecGroupRule) FromCreateRequest(c []byte) (Resource, error) {
	createOpts := struct {
		rules.CreateOpts `json:"security_group_rule"`
	}{}
	err := json.Unmarshal(c, &createOpts)
	if err != nil {
		return nil, err
	}
	s.ID = SecGroupRuleID
	s.Description = createOpts.Description
	s.ProjectID = createOpts.ProjectID
	s.Direction = string(createOpts.Direction)
	s.EtherType = string(createOpts.EtherType)
	s.PortRangeMax = createOpts.PortRangeMax
	s.PortRangeMin = createOpts.PortRangeMin
	return s, nil
}

func (s *SecGroupRule) CreateResponse() ([]byte, error) {
	res := struct {
		SecGroupRule *SecGroupRule `json:"security_group_rule"`
	}{
		SecGroupRule: s,
	}
	return json.Marshal(res)
}

func (s *SecGroupRule) SubResources() []ResourceBuilder {
	return nil
}

type Subnet struct {
	subnets.Subnet
}

func (s Subnet) GetID() string {
	return s.ID
}

func (s Subnet) GetName() string {
	return s.Name
}

func (s Subnet) GetType() string {
	return "subnets"
}

func (s Subnet) GetPath() string {
	return "/subnets"
}

func (s *Subnet) FromCreateRequest(c []byte) (Resource, error) {
	createOpts := struct {
		subnets.CreateOpts `json:"subnet"`
	}{}
	err := json.Unmarshal(c, &createOpts)
	if err != nil {
		return nil, err
	}
	s.ID = SubnetID
	s.Description = createOpts.Description
	s.ProjectID = createOpts.ProjectID
	s.AllocationPools = createOpts.AllocationPools
	s.CIDR = createOpts.CIDR
	s.DNSNameservers = createOpts.DNSNameservers
	if createOpts.GatewayIP != nil {
		s.GatewayIP = *createOpts.GatewayIP
	}
	if createOpts.EnableDHCP != nil {
		s.EnableDHCP = *createOpts.EnableDHCP
	}
	return s, nil
}

func (s *Subnet) CreateResponse() ([]byte, error) {
	res := struct {
		Subnet *Subnet `json:"subnet"`
	}{
		Subnet: s,
	}
	return json.Marshal(res)
}

func (s *Subnet) SubResources() []ResourceBuilder {
	return nil
}

type SubnetPool subnetpools.SubnetPool

func (s SubnetPool) GetID() string {
	return s.ID
}

func (s SubnetPool) GetName() string {
	return s.Name
}

func (s SubnetPool) GetType() string {
	return "subnetpools"
}

func (s SubnetPool) GetPath() string {
	return "/subnetpools"
}

func (s *SubnetPool) FromCreateRequest(c []byte) (Resource, error) {
	createOpts := struct {
		subnetpools.CreateOpts `json:"subnetPool"`
	}{}
	err := json.Unmarshal(c, &createOpts)
	if err != nil {
		return nil, err
	}
	s.ID = SubnetPoolID
	s.Description = createOpts.Description
	s.ProjectID = createOpts.ProjectID
	return s, nil
}

func (s *SubnetPool) CreateResponse() ([]byte, error) {
	res := struct {
		SubnetPool *SubnetPool `json:"subnetPool"`
	}{
		SubnetPool: s,
	}
	return json.Marshal(res)
}

func (s *SubnetPool) SubResources() []ResourceBuilder {
	return nil
}

type Port ports.Port

func (p Port) GetID() string {
	return p.ID
}

func (p Port) GetName() string {
	return p.Name
}

func (p Port) GetType() string {
	return "ports"
}

func (p Port) GetPath() string {
	return "/ports"
}

func (p *Port) FromCreateRequest(c []byte) (Resource, error) {
	createOpts := struct {
		ports.CreateOpts `json:"port"`
	}{}
	err := json.Unmarshal(c, &createOpts)
	if err != nil {
		return nil, err
	}
	p.ID = PortID
	p.Description = createOpts.Description
	p.ProjectID = createOpts.ProjectID
	p.DeviceID = createOpts.DeviceID
	if createOpts.AdminStateUp != nil {
		p.AdminStateUp = *createOpts.AdminStateUp
	}
	return p, nil
}

func (p *Port) CreateResponse() ([]byte, error) {
	res := struct {
		*Port `json:"subnet"`
	}{
		Port: p,
	}
	return json.Marshal(res)
}

func (p *Port) SubResources() []ResourceBuilder {
	return nil
}

type Router routers.Router

func (r Router) GetID() string {
	return r.ID
}

func (r Router) GetName() string {
	return r.Name
}

func (r Router) GetType() string {
	return "routers"
}

func (r Router) GetPath() string {
	return "/routers"
}

func (r *Router) FromCreateRequest(c []byte) (Resource, error) {
	createOpts := struct {
		routers.CreateOpts `json:"router"`
	}{}
	err := json.Unmarshal(c, &createOpts)
	if err != nil {
		return nil, err
	}
	r.ID = RouterID
	r.Description = createOpts.Description
	r.ProjectID = createOpts.ProjectID
	if createOpts.Distributed != nil {
		r.Distributed = *createOpts.Distributed
	}
	if createOpts.GatewayInfo != nil {
		r.GatewayInfo = *createOpts.GatewayInfo
	}
	if createOpts.AdminStateUp != nil {
		r.AdminStateUp = *createOpts.AdminStateUp
	}
	return r, nil
}

func (r *Router) CreateResponse() ([]byte, error) {
	res := struct {
		Router *Router `json:"router"`
	}{
		Router: r,
	}
	return json.Marshal(res)
}

func (r *Router) SubResources() []ResourceBuilder {
	return []ResourceBuilder{
		func() Resource {
			return &InterfaceInfo{routerID: r.ID}
		},
		func() Resource { return &ResourceTags{resourceID: r.ID, resourceType: "routers"} },
	}
}

type ResourceTags struct {
	resourceID   string
	resourceType string
	Tags         []string `json:"tags"`
}

func (t ResourceTags) GetType() string { return "tags" }
func (t ResourceTags) GetID() string   { return t.resourceID }
func (t ResourceTags) GetName() string { return t.resourceID }

func (t ResourceTags) GetPath() string {
	return fmt.Sprintf("/%s/%s/tags", t.resourceType, t.resourceID)
}
func (t *ResourceTags) FromCreateRequest(c []byte) (Resource, error) {
	// expect {"tags": ["foo","bar",...]}
	var body struct {
		Tags []string `json:"tags"`
	}
	if err := json.Unmarshal(c, &body); err != nil {
		return nil, err
	}
	t.Tags = body.Tags
	return t, nil
}

func (t *ResourceTags) CreateResponse() ([]byte, error) {
	return json.Marshal(struct{ Tags []string }{t.Tags})
}
func (t *ResourceTags) SubResources() []ResourceBuilder { return nil }

type InterfaceInfo struct {
	routers.InterfaceInfo
	routerID string
}

func (i InterfaceInfo) GetID() string {
	return i.ID
}

func (i InterfaceInfo) GetName() string {
	return ""
}

func (i InterfaceInfo) GetType() string {
	return "interfacesInfo"
}

func (i InterfaceInfo) GetPath() string {
	return fmt.Sprintf("/routers/%s/add_router_interface", i.routerID)
}

func (i *InterfaceInfo) FromCreateRequest(c []byte) (Resource, error) {
	addOpts := routers.AddInterfaceOpts{}
	err := json.Unmarshal(c, &addOpts)
	if err != nil {
		return nil, err
	}
	i.ID = InterfaceInfoID
	i.PortID = addOpts.PortID
	i.SubnetID = addOpts.SubnetID
	return i, nil
}

func (i *InterfaceInfo) CreateResponse() ([]byte, error) {
	return json.Marshal(i)
}

func (i *InterfaceInfo) SubResources() []ResourceBuilder {
	return nil
}

type Request struct {
	Method string
	Path   string
}

// Simulator provides an OpenStack API server to be used for testing purposes.
type Simulator struct {
	*testing.T
	sync.RWMutex
	resources map[string][]Resource
	builders  map[string]ResourceBuilder
	reqCount  map[Request]int
}

func NewSimulator(t *testing.T) *Simulator {
	// Start listening for incoming requests
	ostesthelper.SetupHTTP()
	// Create the simulator and register resource handlers
	return (&Simulator{
		T:         t,
		builders:  map[string]ResourceBuilder{},
		resources: map[string][]Resource{},
		reqCount:  map[Request]int{},
	}).
		Register(func() Resource { return &Network{} }).
		Register(func() Resource { return &Subnet{} }).
		Register(func() Resource { return &SubnetPool{} }).
		Register(func() Resource { return &SecGroup{} }).
		Register(func() Resource { return &SecGroupRule{} }).
		Register(func() Resource { return &Router{} }).
		Register(func() Resource { return &Port{} })
}

func (s *Simulator) TearDown() {
	ostesthelper.TeardownHTTP()
}

func (s *Simulator) Register(builder ResourceBuilder) *Simulator {
	s.Lock()
	defer s.Unlock()
	path := builder().GetPath()
	s.builders[path] = builder
	ostesthelper.Mux.HandleFunc(path, s.Handle)
	return s
}

func (s *Simulator) Add(res ...Resource) *Simulator {
	for _, r := range res {
		if b, ok := s.builders[r.GetPath()]; !ok || reflect.TypeOf(b()) != reflect.TypeOf(r) {
			panic(fmt.Sprintf("cannot insert resource of type %s", r.GetType()))
		}
		s.resources[r.GetPath()] = append(s.resources[r.GetPath()], r)
		for _, sub := range r.SubResources() {
			s.Register(sub)
		}
	}
	return s
}

func (s *Simulator) GetRequestCounters() map[Request]int {
	s.RLock()
	defer s.RUnlock()
	currCounters := make(map[Request]int, len(s.reqCount))
	for k, v := range s.reqCount {
		currCounters[k] = v
	}
	return currCounters
}

func (s *Simulator) GetClient() *gophercloud.ServiceClient {
	return osfakeclient.ServiceClient()
}

func (s *Simulator) Handle(w http.ResponseWriter, r *http.Request) {
	// Increase req counter
	s.Lock()
	s.reqCount[Request{Path: r.URL.Path, Method: r.Method}]++
	s.Unlock()
	// Handle request
	ostesthelper.TestHeader(s.T, r, "X-Auth-Token", osfakeclient.TokenID)
	switch r.Method {
	case http.MethodGet:
		out, err := func() ([]byte, error) {
			s.RLock()
			defer s.RUnlock()
			typeP := s.builders[r.URL.Path]().GetType()
			name := r.URL.Query().Get("name")
			// Filter by name.
			var filtered []Resource
			for _, r := range s.resources[r.URL.Path] {
				if name != "" && r.GetName() != name {
					continue
				}
				filtered = append(filtered, r)
			}
			return json.Marshal(map[string][]Resource{typeP: filtered})
		}()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, err.Error())
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, string(out))
	case http.MethodPost:
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, err.Error())
		}
		newRes, err := s.builders[r.URL.Path]().FromCreateRequest(reqBody)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, err.Error())
		}
		res, err := newRes.CreateResponse()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, err.Error())
		}
		for _, dh := range newRes.SubResources() {
			s.Register(dh)
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintln(w, string(res))
		s.Lock()
		s.resources[r.URL.Path] = append(s.resources[r.URL.Path], newRes)
		s.Unlock()
	case http.MethodPut:
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, err.Error())
		}
		newRes, err := s.builders[r.URL.Path]().FromCreateRequest(reqBody)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, err.Error())
		}
		res, err := newRes.CreateResponse()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, err.Error())
		}
		for _, dh := range newRes.SubResources() {
			s.Register(dh)
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, string(res))
		s.Lock()
		s.resources[r.URL.Path] = append(s.resources[r.URL.Path], newRes)
		s.Unlock()
	default:
		w.WriteHeader(http.StatusNotImplemented)
	}
}
