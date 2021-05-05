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
	"sort"
	"strings"

	"github.com/pkg/errors"

	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// SortServicesByCreationTimestamp sorts the Service slice in descending order
// by creation timestamp (i.e. from oldest to newest).
// UID alphanumeric ordering is used to break ties.
func SortServicesByCreationTimestamp(items []corev1.Service) {
	sort.Slice(items, func(i, j int) bool {
		if it, jt := items[i].CreationTimestamp, items[j].CreationTimestamp; it != jt {
			return jt.After(it.Time)
		}
		// Break ties with UIDs
		return items[i].UID < items[j].UID
	})
}

// ServiceKey returns a string used to identify the given v1.Service.
func ServiceKey(svc *corev1.Service) string {
	return fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)
}

// ServicePortKey returns a string used to identify the given v1.ServicePort.
func ServicePortKey(serviceKey string, servicePort *corev1.ServicePort) string {
	if servicePort.Name == "" {
		return serviceKey
	}
	return fmt.Sprintf("%s-%s", serviceKey, servicePort.Name)
}

func isExposed(obj metav1.Object, exposeAnnotationKey string) bool {
	return len(extractExposeTypes(obj, exposeAnnotationKey)) > 0
}

func extractExposeTypes(obj metav1.Object, exposeAnnotationKey string) nodeportproxy.ExposeTypes {
	res := nodeportproxy.NewExposeTypes()
	if obj.GetAnnotations() == nil {
		return res
	}
	// When legacy value 'true' is encountered NodePortType is returned for
	// backward compatibility.
	val := obj.GetAnnotations()[exposeAnnotationKey]
	if val == "true" {
		res.Insert(nodeportproxy.NodePortType)
		return res
	}
	// Parse the comma separated list and return the list of ExposeType
	ts := strings.Split(val, ",")
	for i := range ts {
		t, ok := nodeportproxy.ExposeTypeFromString(strings.TrimSpace(ts[i]))
		if !ok {
			// If we met a not valid token we consider the value invalid and
			// return an empty set.
			return nodeportproxy.NewExposeTypes()
		}
		res.Insert(t)
	}
	return res
}

// portNamesSet returns the set of port names extracted from the given Service.
func portNamesSet(svc *corev1.Service, filterFuncs ...func(corev1.ServicePort) bool) sets.String {
	portNames := sets.NewString()
OUTER:
	for _, p := range svc.Spec.Ports {
		for _, filter := range filterFuncs {
			if ok := filter(p); !ok {
				continue OUTER
			}
		}
		portNames.Insert(p.Name)
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
	val, ok := a[nodeportproxy.PortHostMappingAnnotationKey]
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
	tcpPortNames := portNamesSet(svc, func(p corev1.ServicePort) bool { return p.Protocol == corev1.ProtocolTCP })
	if diff := portNames.Difference(tcpPortNames); len(diff) > 0 {
		return fmt.Errorf("port name(s) not found in TCP service ports: %v", diff.List())
	}
	return nil
}
