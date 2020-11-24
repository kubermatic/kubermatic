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

// ExposeType defines the strategy used to expose the service.
type ExposeType string

func ExposeTypeFromString(s string) (ExposeType, bool) {
	switch ExposeType(s) {
	case NodePortType:
		return NodePortType, true
	case SNIType:
		return SNIType, true
	case HTTP2ConnectType:
		return HTTP2ConnectType, true
	default:
		return NodePortType, false
	}
}

const (
	// NodePortType is the default ExposeType which creates a listener for each
	// NodePort.
	NodePortType ExposeType = "NodePort"
	// SNIType configures Envoy to route TLS streams based on SNI
	// without terminating them.
	SNIType = "SNI"
	// HTTP2ConnectType configures Envoy to terminate HTTP/2 Connect requests.
	HTTP2ConnectType = "HTTP2Connect"
)

const (
	DefaultExposeAnnotationKey = "nodeport-proxy.k8s.io/expose"
	// PortHostMappingAnnotationKey contains the mapping between the port to be
	// exposed and the hostname, this is only used when the ExposeType is
	// SNIType.
	PortHostMappingAnnotationKey = "nodeport-proxy.k8s.io/port-mapping"
)

func ServiceKey(service *corev1.Service) string {
	return fmt.Sprintf("%s/%s", service.Namespace, service.Name)
}

func isExposed(obj metav1.Object, exposeAnnotationKey string) bool {
	return len(extractExposeTypes(obj, exposeAnnotationKey)) > 0
}

func (e ExposeType) String() string {
	return string(e)
}

func extractExposeTypes(obj metav1.Object, exposeAnnotationKey string) []ExposeType {
	var types []ExposeType
	if obj.GetAnnotations() == nil {
		return types
	}
	// When legacy value 'true' is encountered NodePortType is returned for
	// backward compatibility.
	val := obj.GetAnnotations()[exposeAnnotationKey]
	if val == "true" {
		return append(types, NodePortType)
	}
	// Parse the comma separated list and return the list of ExposeType
	ts := strings.Split(val, ",")
	for i := range ts {
		ts[i] = strings.TrimSpace(ts[i])
	}
	for _, t := range sets.NewString(ts...).List() {
		e, ok := ExposeTypeFromString(t)
		if !ok {
			return nil
		}
		types = append(types, e)
	}
	return types
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

type portHostMapping map[string]string

func portHostMappingFromService(svc *corev1.Service) (portHostMapping, error) {
	m := portHostMapping{}
	a := svc.GetAnnotations()
	if a == nil {
		return m, fmt.Errorf("service %s/%s does not contain port mapping annotation", svc.Namespace, svc.Name)
	}
	val := a[PortHostMappingAnnotationKey]
	err := json.Unmarshal([]byte(val), m)
	if err != nil {
		return m, errors.Wrap(err, "failed to unmarshal port host mapping for service")
	}
	return m, nil
}

func (p portHostMapping) validate(svc *corev1.Service) error {
	portNames, hosts := p.portHostSets()
	if len(p) > hosts.Len() {
		return fmt.Errorf("duplicated hostname in port host mapping of service: %v", hosts.List())
	}
	actualPortNames := portNamesSet(svc)
	if diff := portNames.Difference(actualPortNames); len(diff) > 0 {
		return fmt.Errorf("ports declared in port host mapping not found in service: %v", diff.List())
	}
	return nil
}

func (p portHostMapping) portHostSets() (sets.String, sets.String) {
	hosts := sets.NewString()
	portNames := sets.NewString()
	for portName, host := range p {
		hosts.Insert(host)
		portNames.Insert(portName)
	}
	return portNames, hosts
}
