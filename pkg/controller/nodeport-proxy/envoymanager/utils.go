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

package envoymanager

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	DefaultExposeAnnotationKey = "nodeport-proxy.k8s.io/expose"
	// PortHostMappingAnnotationKey contains the mapping between the port to be
	// exposed and the hostname, this is only used when the ExposeType is
	// SNIType.
	PortHostMappingAnnotationKey = "nodeport-proxy.k8s.io/port-mapping"
)

// ExposeType defines the strategy used to expose the service.
type ExposeType int

const (
	// NodePortType is the default ExposeType which creates a listener for each
	// NodePort.
	NodePortType ExposeType = iota
	// SNIType configures Envoy to route TLS streams based on SNI
	// without terminating them.
	SNIType
	// HTTP2ConnectType configures Envoy to terminate HTTP/2 Connect requests.
	HTTP2ConnectType
)

// exposeTypeStrings contains the string representation of the ExposeTypes.
var exposeTypeStrings = [...]string{"NodePort", "SNI", "HTTP2Connect"}

// ExposeTypeFromString returns the ExposeType which string representation
// corresponds to the input string, and a boolean indicating whether the
// corresponding ExposeType was found or not.
func ExposeTypeFromString(s string) (ExposeType, bool) {
	switch s {
	case exposeTypeStrings[NodePortType]:
		return NodePortType, true
	case exposeTypeStrings[SNIType]:
		return SNIType, true
	case exposeTypeStrings[HTTP2ConnectType]:
		return HTTP2ConnectType, true
	default:
		return NodePortType, false
	}
}

// String returns the string representation of the ExposeType.
func (e ExposeType) String() string {
	return exposeTypeStrings[e]
}

type ExposeTypes map[ExposeType]sets.Empty

func NewExposeTypes(exposeTypes ...ExposeType) ExposeTypes {
	ets := ExposeTypes{}
	for _, et := range exposeTypes {
		ets[et] = sets.Empty{}
	}
	return ets
}

func (e ExposeTypes) Has(item ExposeType) bool {
	_, contained := e[item]
	return contained
}

func (e ExposeTypes) Insert(item ExposeType) {
	e[item] = sets.Empty{}
}

// ServiceKey returns a string used to identify the given Service.
func ServiceKey(service *corev1.Service) string {
	return fmt.Sprintf("%s/%s", service.Namespace, service.Name)
}

// ServicePortKey returns a string used to identify the given ServicePort.
func ServicePortKey(serviceKey string, servicePort *corev1.ServicePort) string {
	if servicePort.Name == "" {
		return serviceKey
	}
	return fmt.Sprintf("%s-%s", serviceKey, servicePort.Name)
}

func isExposed(obj metav1.Object, exposeAnnotationKey string) bool {
	return len(extractExposeTypes(obj, exposeAnnotationKey)) > 0
}

func extractExposeTypes(obj metav1.Object, exposeAnnotationKey string) ExposeTypes {
	res := NewExposeTypes()
	if obj.GetAnnotations() == nil {
		return res
	}
	// When legacy value 'true' is encountered NodePortType is returned for
	// backward compatibility.
	val := obj.GetAnnotations()[exposeAnnotationKey]
	if val == "true" {
		res.Insert(NodePortType)
		return res
	}
	// Parse the comma separated list and return the list of ExposeType
	ts := strings.Split(val, ",")
	for i := range ts {
		t, ok := ExposeTypeFromString(strings.TrimSpace(ts[i]))
		if !ok {
			// If we met a not valid token we consider the value invalid and
			// return an empty set.
			return NewExposeTypes()
		}
		res.Insert(t)
	}
	return res
}

// portNamesSet returns the set of port names extracted from the given Service.
func portNamesSet(svc *corev1.Service) sets.String {
	portNames := sets.NewString()
	for _, p := range svc.Spec.Ports {
		if p.Protocol == corev1.ProtocolTCP {
			portNames.Insert(p.Name)
		}
	}
	return portNames
}

// portHostSets returns respectively the set of port names and hosts from the
// portHostMapping.
func (p portHostMapping) portHostSets() (sets.String, sets.String) {
	hosts := sets.NewString()
	portNames := sets.NewString()
	for portName, host := range p {
		hosts.Insert(host)
		portNames.Insert(portName)
	}
	return portNames, hosts
}

// portHostMapping contains the mapping between port name and hostname, used
// for SNI ExposeType.
type portHostMapping map[string]string

func portHostMappingFromAnnotation(svc *corev1.Service) (portHostMapping, error) {
	m := portHostMapping{}
	a := svc.GetAnnotations()
	if a == nil {
		return m, nil
	}
	val, ok := a[PortHostMappingAnnotationKey]
	if !ok {
		return m, nil
	}
	err := json.Unmarshal([]byte(val), &m)
	if err != nil {
		return m, errors.Wrap(err, "failed to unmarshal port host mapping")
	}
	return m, nil
}

func (p portHostMapping) validate(svc *corev1.Service) error {
	// TODO(irozzo): validate that hosts are well formed FQDN
	portNames, hosts := p.portHostSets()
	if len(p) > hosts.Len() {
		return fmt.Errorf("duplicated hostname in port host mapping of service: %v", p)
	}
	actualPortNames := portNamesSet(svc)
	if diff := portNames.Difference(actualPortNames); len(diff) > 0 {
		return fmt.Errorf("ports declared in port host mapping not found in service: %v", diff.List())
	}
	return nil
}
