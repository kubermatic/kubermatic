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

package dualstack

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/flagopts"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	"k8c.io/machine-controller/sdk/net"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/clientcmd"
	netutils "k8s.io/utils/net"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	logOptions              = utils.DefaultLogOptions
	enabledOperatingSystems = sets.New[string]()
	enabledCNIs             = sets.New[string]()
	enabledProviders        = sets.New[string]()

	userconfig             string
	ipFamily               string
	skipNodes              bool
	skipHostNetworkPods    bool
	skipEgressConnectivity bool
)

func init() {
	flag.StringVar(&userconfig, "userconfig", "", "path to kubeconfig of usercluster (only used when running TestExistingCluster test)")
	flag.StringVar(&ipFamily, "ipFamily", "IPv4", "IP family")
	flag.Var(flagopts.SetFlag(enabledOperatingSystems), "os", "Comma-separated list of operating systems to test, like ubuntu,flatcar")
	flag.Var(flagopts.SetFlag(enabledCNIs), "cni", "Comma-separated list of CNIs, like cilium,canal")
	flag.Var(flagopts.SetFlag(enabledProviders), "provider", "Comma-separated list of cloud providers, like azure,aws,gcp")
	flag.BoolVar(&skipNodes, "skip-nodes", false, "If true, skips node IP address tests")
	flag.BoolVar(&skipHostNetworkPods, "skip-host-network-pods", false, "If true, skips host network pods IP test")
	flag.BoolVar(&skipEgressConnectivity, "skip-egress-connectivity", false, "If true, skips egress connectivity test")

	alibabaCredentials.AddFlags(flag.CommandLine)
	awsCredentials.AddFlags(flag.CommandLine)
	azureCredentials.AddFlags(flag.CommandLine)
	digitaloceanCredentials.AddFlags(flag.CommandLine)
	equinixMetalCredentials.AddFlags(flag.CommandLine)
	gcpCredentials.AddFlags(flag.CommandLine)
	hetznerCredentials.AddFlags(flag.CommandLine)
	openstackCredentials.AddFlags(flag.CommandLine)
	vsphereCredentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

// TestExistingCluster is used to run dualstack test against any existing cluster.
// Takes kubeconfig of the cluster as command line argument.
func TestExistingCluster(t *testing.T) {
	ctx := context.Background()
	log := log.NewFromOptions(logOptions).Sugar()

	// based on https://kubernetes.io/docs/tasks/network/validate-dual-stack/
	if userconfig == "" {
		t.Logf("kubeconfig for usercluster not provided, test passes vacuously.")
		t.Logf("to run against ready usercluster use following command")
		t.Logf("go test ./pkg/test/dualstack -v -race -tags dualstack -timeout 30m -run TestExistingCluster -args --userconfig <USERCLUSTER KUBECONFIG> --ipFamily <IP FAMILY>")
		return
	}

	config, err := clientcmd.BuildConfigFromFlags("", userconfig)
	if err != nil {
		t.Fatalf("failed to build config: %s", err)
	}

	userclusterClient, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		t.Fatalf("failed to create usercluster client: %s", err)
	}

	testUserCluster(t, ctx, log, userclusterClient, net.IPFamily(ipFamily), skipNodes, skipHostNetworkPods, skipEgressConnectivity)
}

func testUserCluster(t *testing.T, ctx context.Context, log *zap.SugaredLogger, userclusterClient ctrlruntimeclient.Client, ipFamily net.IPFamily, skipNodes, skipHostNetworkPods, skipEgressConnectivity bool) {
	log.Infow("Testing cluster", "ipfamily", ipFamily)

	// get events from user-cluster for debugging
	// aws+canal e2e flakes
	// TODO: check again in 1m; remove if the test looks good
	defer func() {
		events := new(corev1.EventList)
		err := userclusterClient.List(ctx, events)
		if err != nil {
			t.Logf("Failed to get events from usercluster: %+v", err)
		}
		t.Log("Events for debugging logged below.")
		for _, event := range events.Items {
			e, _ := json.Marshal(event)
			t.Log(string(e))
		}
	}()

	// validate nodes
	if skipNodes {
		log.Info("Skipping validation for nodes")
	} else {
		nodes := corev1.NodeList{}
		if err := userclusterClient.List(ctx, &nodes); err != nil {
			t.Fatalf("Failed to list nodes: %v", err)
		}

		for _, node := range nodes.Items {
			var addrs []string
			for _, addr := range node.Status.Addresses {
				if addr.Type == corev1.NodeHostName {
					continue
				}
				addrs = append(addrs, addr.Address)
			}
			validate(t, fmt.Sprintf("node '%s' status addresses", node.Name), ipFamily, addrs)
		}

		for _, node := range nodes.Items {
			if len(node.Spec.PodCIDRs) > 0 {
				// in case of Cilium we can have 0 pod CIDRs as Cilium uses its own node IPAM
				validate(t, fmt.Sprintf("node '%s' pod CIDRs", node.Name), ipFamily, node.Spec.PodCIDRs)
			}
		}
	}

	nodes := corev1.NodeList{}
	if err := userclusterClient.List(ctx, &nodes); err != nil {
		t.Fatalf("Failed to list nodes: %v", err)
	}

	nNodes := len(nodes.Items)

	// validate pods
	pods := corev1.PodList{}
	if err := userclusterClient.List(ctx, &pods, ctrlruntimeclient.InNamespace(metav1.NamespaceSystem)); err != nil {
		t.Fatalf("Failed to list pods: %v", err)
	}

	for _, pod := range pods.Items {
		if pod.Spec.HostNetwork && skipHostNetworkPods {
			log.Infow("skipping host network pod", "pod", pod.Name)
			continue
		}

		var podAddrs []string
		for _, addr := range pod.Status.PodIPs {
			podAddrs = append(podAddrs, addr.IP)
		}
		validate(t, fmt.Sprintf("pod '%s' addresses", pod.Name), ipFamily, podAddrs)
	}

	// validate services
	services := corev1.ServiceList{}
	if err := userclusterClient.List(ctx, &services, ctrlruntimeclient.InNamespace(metav1.NamespaceSystem)); err != nil {
		t.Fatalf("Failed to list services: %v", err)
	}

	for _, svc := range services.Items {
		svcLog := log.With("service", svc.Name)

		if svc.Spec.IPFamilyPolicy == nil {
			svcLog.Info("Skipping because Spec.IPFamilyPolicy is not set")
			continue
		}

		switch *svc.Spec.IPFamilyPolicy {
		case corev1.IPFamilyPolicySingleStack:
			if ipFamily == net.IPFamilyIPv4IPv6 {
				svcLog.Infof("Skipping %q test because IP family policy is %q", ipFamily, *svc.Spec.IPFamilyPolicy)
				continue
			}
		case corev1.IPFamilyPolicyPreferDualStack, corev1.IPFamilyPolicyRequireDualStack:
		}

		switch svc.Spec.Type {
		case corev1.ServiceTypeClusterIP:
			validate(t, fmt.Sprintf("service '%s' cluster IPs", svc.Name), ipFamily, svc.Spec.ClusterIPs)
		case corev1.ServiceTypeNodePort:
		case corev1.ServiceTypeExternalName:
		case corev1.ServiceTypeLoadBalancer:
			validate(t, fmt.Sprintf("service '%s' cluster IPs", svc.Name), ipFamily, svc.Spec.ClusterIPs)
			validate(t, fmt.Sprintf("service '%s' external IPs", svc.Name), ipFamily, svc.Spec.ExternalIPs)
		}
	}

	// validate egress connectivity
	if skipEgressConnectivity {
		log.Info("Skipping validation of egress connectivity")
	} else {
		switch ipFamily {
		case net.IPFamilyIPv4, net.IPFamilyUnspecified:
			validateEgressConnectivity(t, ctx, log, userclusterClient, 4, nNodes)
		case net.IPFamilyIPv6:
			validateEgressConnectivity(t, ctx, log, userclusterClient, 6, nNodes)
		case net.IPFamilyIPv4IPv6:
			validateEgressConnectivity(t, ctx, log, userclusterClient, 4, nNodes)
			validateEgressConnectivity(t, ctx, log, userclusterClient, 6, nNodes)
		}
	}
}

func validateEgressConnectivity(t *testing.T, ctx context.Context, log *zap.SugaredLogger, userclusterClient ctrlruntimeclient.Client, ipVersion, expectedPodCount int) {
	log.Infof("validating %s", fmt.Sprintf("egress-validator-%d", ipVersion))

	ds := egressValidatorDaemonSet(ipVersion, metav1.NamespaceDefault)
	if err := userclusterClient.Create(ctx, ds); err != nil {
		t.Errorf("failed to create DaemonSet: %v", err)
		return
	}

	defer func() {
		if err := userclusterClient.Delete(ctx, ds); err != nil {
			t.Errorf("failed to cleanup: %v", err)
			return
		}
	}()

	err := wait.PollLog(ctx, log, 10*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		d := &appsv1.DaemonSet{}
		if err := userclusterClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(ds), d); err != nil {
			return fmt.Errorf("failed to get DaemonSet: %w", err), nil
		}

		if int(d.Status.NumberAvailable) != expectedPodCount {
			return fmt.Errorf("only %d out of %d pods available", d.Status.NumberAvailable, expectedPodCount), nil
		}

		return nil, nil
	})
	if err != nil {
		t.Errorf("DaemonSet never became healthy: %v", err)
	}
}

func validate(t *testing.T, name string, ipFamily net.IPFamily, addrs []string) {
	if !all(ipFamily, addrs) {
		t.Errorf("not all addresses in %s are in IP family %q for %s", addrs, ipFamily, name)
	}
}

func all(ipFamily net.IPFamily, addrs []string) bool {
	// We convert all the IPs to CIDR notation so that we can simply use CIDR
	// validation functions everywhere instead of checking which function
	// to use every time.
	// Actual length of the mask doesn't matter, so it is set to 0.
	for i, addr := range addrs {
		if !strings.Contains(addr, "/") {
			addrs[i] = fmt.Sprintf("%s/0", addr)
		}
	}

	switch ipFamily {
	case net.IPFamilyIPv4, net.IPFamilyUnspecified:
		for _, addr := range addrs {
			if !netutils.IsIPv4CIDRString(addr) {
				return false
			}
		}
	case net.IPFamilyIPv6:
		for _, addr := range addrs {
			if !netutils.IsIPv6CIDRString(addr) {
				return false
			}
		}
	case net.IPFamilyIPv4IPv6:
		ok, err := netutils.IsDualStackCIDRStrings(addrs)
		return err == nil && ok
	default:
		return false
	}

	return true
}

func egressValidatorDaemonSet(ipVersion int, namespace string) *appsv1.DaemonSet {
	pod := egressValidatorPod(ipVersion)
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("egress-validator-%d", ipVersion),
			Namespace: namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": fmt.Sprintf("egress-validator-%d", ipVersion),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: pod.ObjectMeta,
				Spec:       pod.Spec,
			},
		},
	}
}

func egressValidatorPod(ipVersion int) *corev1.Pod {
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("egress-validator-%d", ipVersion),
			Labels: map[string]string{
				"name": fmt.Sprintf("egress-validator-%d", ipVersion),
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  fmt.Sprintf("egress-validator-%d-container", ipVersion),
					Image: "docker.io/byrnedo/alpine-curl:0.1.8",
					Command: []string{
						"/bin/ash",
						"-c",
						"while true; do sleep 1; done",
					},
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							Exec: &corev1.ExecAction{
								Command: []string{
									"curl",
									"-sS",
									fmt.Sprintf("--ipv%d", ipVersion),
									"--fail",
									"--connect-timeout",
									"5",
									"-o",
									"/dev/null",
									fmt.Sprintf("v%d.ident.me", ipVersion),
								},
							},
						},
						TimeoutSeconds: 7,
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							Exec: &corev1.ExecAction{
								Command: []string{
									"curl",
									"-sS",
									fmt.Sprintf("--ipv%d", ipVersion),
									"--fail",
									"--connect-timeout",
									"5",
									"-o",
									"/dev/null",
									fmt.Sprintf("v%d.ident.me", ipVersion),
								},
							},
						},
						TimeoutSeconds: 7,
					},
				},
			},
			HostNetwork: false,
		},
	}
}
