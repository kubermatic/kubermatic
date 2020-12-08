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

package v1

// ExposeStrategy is the strategy to expose the cluster with.
type ExposeStrategy string

const (
	// ExposeStrategyNodePort creates a NodePort with a "nodeport-proxy.k8s.io/expose": "true" annotation to expose
	// all clusters on one central Service of type LoadBalancer via the NodePort proxy.
	ExposeStrategyNodePort ExposeStrategy = "NodePort"
	// ExposeStrategyLoadBalancer creates a LoadBalancer service per cluster.
	ExposeStrategyLoadBalancer ExposeStrategy = "LoadBalancer"
	// ExposeStrategyTunneling allows to reach the control plane components by
	// tunneling L4 traffic (TCP only is supported at the moment).
	// The traffic is encapsulated by mean of an agent deployed on the worker
	// nodes.
	// The traffic is decapsulated and forwarded to the recipients by
	// mean of a proxy deployed on the Seed Cluster.
	// The same proxy is also capable of routing TLS traffic without
	// termination, this is to allow the Kubelet to reach the control plane
	// before the agents are running.
	//
	// This strategy has the inconvenient of requiring an agent on worker
	// nodes, but has the notable advantage of requiring one single entry point
	// (e.g. Service of type LoadBalancer) without consuming one or more ports
	// for each user cluster.
	ExposeStrategyTunneling ExposeStrategy = "Tunneling"
)
