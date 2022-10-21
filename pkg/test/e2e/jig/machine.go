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

package jig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	awstypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/aws/types"
	hetznertypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/hetzner/types"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	"k8c.io/operating-system-manager/pkg/providerconfig/amzn2"
	"k8c.io/operating-system-manager/pkg/providerconfig/centos"
	"k8c.io/operating-system-manager/pkg/providerconfig/rhel"
	"k8c.io/operating-system-manager/pkg/providerconfig/rockylinux"
	"k8c.io/operating-system-manager/pkg/providerconfig/sles"
	"k8c.io/operating-system-manager/pkg/providerconfig/ubuntu"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type MachineJig struct {
	client ctrlruntimeclient.Client
	log    *zap.SugaredLogger

	// Either specify the cluster directly or specify the clusterJig,
	// if at the time of creating the MachineJig the cluster doesn't exist yet.
	cluster    *kubermaticv1.Cluster
	clusterJig *ClusterJig

	// user-controller parameters
	name          string
	replicas      int
	osSpec        interface{}
	providerSpec  interface{}
	clusterClient ctrlruntimeclient.Client
}

func NewMachineJig(client ctrlruntimeclient.Client, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) *MachineJig {
	return &MachineJig{
		client:   client,
		log:      log,
		cluster:  cluster,
		name:     "e2e-workers",
		osSpec:   ubuntu.Config{},
		replicas: 1,
	}
}

func (j *MachineJig) WithCluster(cluster *kubermaticv1.Cluster) *MachineJig {
	j.cluster = cluster
	j.clusterJig = nil
	return j
}

func (j *MachineJig) WithClusterJig(jig *ClusterJig) *MachineJig {
	j.clusterJig = jig
	j.cluster = nil
	return j
}

func (j *MachineJig) WithName(name string) *MachineJig {
	j.name = name
	return j
}

func (j *MachineJig) WithReplicas(replicas int) *MachineJig {
	j.replicas = replicas
	return j
}

// WithOSSpec expects arguments like pkg/providerconfig/ubuntu.Config{}.
// Do not use pointers.
func (j *MachineJig) WithOSSpec(spec interface{}) *MachineJig {
	j.osSpec = spec
	return j
}

func (j *MachineJig) WithUbuntu() *MachineJig {
	return j.WithOSSpec(ubuntu.Config{})
}

func (j *MachineJig) WithCentOS() *MachineJig {
	return j.WithOSSpec(centos.Config{})
}

func (j *MachineJig) WithRHEL() *MachineJig {
	return j.WithOSSpec(rhel.Config{})
}

// WithProviderSpec expects arguments like pkg/cloudprovider/provider/aws/types/RawConfig{}.
// Do not use pointers.
func (j *MachineJig) WithProviderSpec(spec interface{}) *MachineJig {
	j.providerSpec = spec
	return j
}

// If you already have a cluster client, you can set it with WithClusterClient().
// Otherwise the MachineJig will retrieve a proper client itself.
func (j *MachineJig) WithClusterClient(client ctrlruntimeclient.Client) *MachineJig {
	j.clusterClient = client
	return j
}

func (j *MachineJig) WithAWS(instanceType string, spotMaxPriceUSD *string) *MachineJig {
	cfg := awstypes.RawConfig{
		InstanceType: providerconfig.ConfigVarString{Value: instanceType},
	}

	if spotMaxPriceUSD != nil {
		cfg.IsSpotInstance = pointer.Bool(true)
		cfg.SpotInstanceConfig = &awstypes.SpotInstanceConfig{
			MaxPrice: providerconfig.ConfigVarString{Value: *spotMaxPriceUSD},
		}
	}

	return j.WithProviderSpec(cfg)
}

func (j *MachineJig) WithHetzner(instanceSize string) *MachineJig {
	return j.WithProviderSpec(hetznertypes.RawConfig{
		ServerType: providerconfig.ConfigVarString{Value: instanceSize},
	})
}

type MachineWaitMode string

const (
	WaitForNothing    MachineWaitMode = ""
	WaitForReadyNodes MachineWaitMode = "nodes"
	WaitForReadyPods  MachineWaitMode = "pods"
)

func (j *MachineJig) Create(ctx context.Context, waitMode MachineWaitMode) error {
	j.log.Infow("Creating MachineDeployment...", "name", j.name, "replicas", j.replicas)

	cluster, err := j.getCluster(ctx)
	if err != nil {
		return fmt.Errorf("failed to determine user cluster: %w", err)
	}

	provider, err := j.determineCloudProvider(cluster)
	if err != nil {
		return fmt.Errorf("failed to determine cloud provider: %w", err)
	}

	os, err := j.determineOperatingSystem()
	if err != nil {
		return fmt.Errorf("failed to determine operating system: %w", err)
	}

	_, datacenter, err := Seed(ctx, j.client)
	if err != nil {
		return fmt.Errorf("failed to determine target datacenter: %w", err)
	}

	providerSpec, err := j.enrichProviderSpec(cluster, provider, datacenter)
	if err != nil {
		return fmt.Errorf("failed to apply cluster information to the provider spec: %w", err)
	}

	encodedCloudProviderSpec, err := json.Marshal(providerSpec)
	if err != nil {
		return fmt.Errorf("failed to encode provider spec: %w", err)
	}

	encodedOSSpec, err := json.Marshal(j.osSpec)
	if err != nil {
		return fmt.Errorf("failed to encode OS spec: %w", err)
	}

	cfg := providerconfig.Config{
		CloudProvider: provider,
		CloudProviderSpec: runtime.RawExtension{
			Raw: encodedCloudProviderSpec,
		},
		OperatingSystem: os,
		OperatingSystemSpec: runtime.RawExtension{
			Raw: encodedOSSpec,
		},
	}

	encodedConfig, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to encode provider config: %w", err)
	}

	labels := map[string]string{
		"md-name": j.name,
	}

	md := clusterv1alpha1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      j.name,
			Namespace: metav1.NamespaceSystem,
		},
		Spec: clusterv1alpha1.MachineDeploymentSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: labels,
			},
			Replicas: pointer.Int32(int32(j.replicas)),
			Template: clusterv1alpha1.MachineTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: clusterv1alpha1.MachineSpec{
					Versions: clusterv1alpha1.MachineVersionInfo{
						Kubelet: cluster.Spec.Version.String(),
					},
					ProviderSpec: clusterv1alpha1.ProviderSpec{
						Value: &runtime.RawExtension{
							Raw: encodedConfig,
						},
					},
				},
			},
		},
	}

	clusterClient, err := j.getClusterClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get cluster client: %w", err)
	}

	utilruntime.Must(clusterv1alpha1.AddToScheme(clusterClient.Scheme()))

	err = wait.PollImmediate(ctx, 1*time.Second, 2*time.Minute, func() (error, error) {
		return clusterClient.Create(ctx, &md), nil
	})
	if err != nil {
		return fmt.Errorf("failed to create MachineDeployment: %w", err)
	}

	j.log.Info("MachineDeployment created successfully.")

	if waitMode == WaitForReadyNodes || waitMode == WaitForReadyPods {
		j.log.Info("Waiting for nodes to become ready...")
		if err = j.waitForReadyNodes(ctx, clusterClient); err != nil {
			return fmt.Errorf("failed to wait: %w", err)
		}
	}

	if waitMode == WaitForReadyPods {
		j.log.Info("Waiting for pods to become ready...")
		if err = j.waitForReadyPods(ctx, clusterClient); err != nil {
			return fmt.Errorf("failed to wait: %w", err)
		}
	}

	return nil
}

func (j *MachineJig) waitForReadyNodes(ctx context.Context, clusterClient ctrlruntimeclient.Client) error {
	return wait.PollLog(ctx, j.log, 10*time.Second, 15*time.Minute, func() (error, error) {
		nodeList := corev1.NodeList{}
		err := clusterClient.List(ctx, &nodeList)
		if err != nil {
			return fmt.Errorf("failed to list nodes: %w", err), nil
		}

		readyNodeCount := 0
		for _, node := range nodeList.Items {
			for _, c := range node.Status.Conditions {
				if c.Type == corev1.NodeReady {
					readyNodeCount++
				}
			}
		}

		if readyNodeCount != j.replicas {
			return fmt.Errorf("%d of %d nodes are ready", readyNodeCount, j.replicas), nil
		}

		return nil, nil
	})
}

func (j *MachineJig) waitForReadyPods(ctx context.Context, clusterClient ctrlruntimeclient.Client) error {
	return wait.PollLog(ctx, j.log, 5*time.Second, 5*time.Minute, func() (error, error) {
		pods := corev1.PodList{}
		if err := clusterClient.List(ctx, &pods); err != nil {
			return fmt.Errorf("failed to list pods: %w", err), nil
		}

		if len(pods.Items) == 0 {
			return errors.New("no pods found"), nil
		}

		unready := sets.NewString()
		for _, pod := range pods.Items {
			if !podIsReadyOrCompleted(&pod) {
				unready.Insert(pod.Name)
			}
		}

		if unready.Len() > 0 {
			return fmt.Errorf("not all pods are ready: %v", unready.List()), nil
		}

		return nil, nil
	})
}

func podIsReadyOrCompleted(pod *corev1.Pod) bool {
	for _, cs := range pod.Status.ContainerStatuses {
		if !containerIsReadyOrCompleted(cs) {
			return false
		}
	}

	for _, cs := range pod.Status.InitContainerStatuses {
		if !containerIsReadyOrCompleted(cs) {
			return false
		}
	}

	return true
}

func containerIsReadyOrCompleted(cs corev1.ContainerStatus) bool {
	if cs.Ready {
		return true
	}

	if cs.State.Terminated != nil && cs.State.Terminated.ExitCode == 0 {
		return true
	}

	return false
}

func (j *MachineJig) Delete(ctx context.Context, synchronous bool) error {
	log := j.log.With("name", j.name)
	log.Info("Deleting MachineDeployment...")

	cluster, err := j.getCluster(ctx)
	if err != nil {
		return fmt.Errorf("failed to determine user cluster: %w", err)
	}

	clusterClient, err := j.getClusterClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get cluster client: %w", err)
	}

	md := clusterv1alpha1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      j.name,
			Namespace: metav1.NamespaceSystem,
		},
	}

	if err := clusterClient.Delete(ctx, &md); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete MachineDeployment: %w", err)
	}

	if synchronous {
		log.Info("Waiting for MachineDeployment to be gone...")

		key := ctrlruntimeclient.ObjectKeyFromObject(&md)
		err = wait.PollLog(ctx, log, 5*time.Second, 10*time.Minute, func() (transient error, terminal error) {
			err := clusterClient.Get(ctx, key, &md)
			if err == nil {
				return errors.New("MachineDeployment still exists"), nil
			}
			if !apierrors.IsNotFound(err) {
				return nil, err
			}

			return nil, nil
		})

		if err != nil {
			return fmt.Errorf("failed to wait for MachineDeployment to be gone: %w", err)
		}
	}

	return nil
}

func (j *MachineJig) getCluster(ctx context.Context) (*kubermaticv1.Cluster, error) {
	if j.clusterJig != nil {
		return j.clusterJig.Cluster(ctx)
	}

	return j.cluster, nil
}

func (j *MachineJig) getClusterClient(ctx context.Context, cluster *kubermaticv1.Cluster) (ctrlruntimeclient.Client, error) {
	if j.clusterClient != nil {
		return j.clusterClient, nil
	}

	projectName := cluster.Labels[kubermaticv1.ProjectIDLabelKey]

	return NewClusterJig(j.client, j.log).
		WithExistingCluster(cluster.Name).
		WithProjectName(projectName).
		ClusterClient(ctx)
}

func (j *MachineJig) determineCloudProvider(cluster *kubermaticv1.Cluster) (providerconfig.CloudProvider, error) {
	provider := providerconfig.CloudProvider(cluster.Spec.Cloud.ProviderName)

	for _, allowed := range providerconfig.AllCloudProviders {
		if allowed == provider {
			return allowed, nil
		}
	}

	return "", fmt.Errorf("unknown cloud provider %q given in cluster cloud spec", cluster.Spec.Cloud.ProviderName)
}

func (j *MachineJig) determineOperatingSystem() (providerconfig.OperatingSystem, error) {
	switch j.osSpec.(type) {
	case centos.Config:
		return providerconfig.OperatingSystemCentOS, nil
	case rhel.Config:
		return providerconfig.OperatingSystemRHEL, nil
	case rockylinux.Config:
		return providerconfig.OperatingSystemRockyLinux, nil
	case sles.Config:
		return providerconfig.OperatingSystemSLES, nil
	case ubuntu.Config:
		return providerconfig.OperatingSystemUbuntu, nil
	case amzn2.Config:
		return providerconfig.OperatingSystemAmazonLinux2, nil
	}

	return "", errors.New("cannot determine OS from the given osSpec")
}

func (j *MachineJig) clusterTags(cluster *kubermaticv1.Cluster) map[string]string {
	name := cluster.Name

	return map[string]string{
		"kubernetes.io/cluster/" + name: "",
		"system/cluster":                name,
	}
}

func (j *MachineJig) enrichProviderSpec(cluster *kubermaticv1.Cluster, provider providerconfig.CloudProvider, datacenter *kubermaticv1.Datacenter) (interface{}, error) {
	switch provider {
	case providerconfig.CloudProviderAWS:
		return j.enrichAWSProviderSpec(cluster, datacenter.Spec.AWS)
	case providerconfig.CloudProviderHetzner:
		return j.enrichHetznerProviderSpec(datacenter.Spec.Hetzner)
	default:
		return nil, fmt.Errorf("don't know how to handle %q provider specs", provider)
	}
}

func (j *MachineJig) enrichAWSProviderSpec(cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecAWS) (interface{}, error) {
	awsConfig, ok := j.providerSpec.(awstypes.RawConfig)
	if !ok {
		return nil, fmt.Errorf("cluster uses AWS, but given provider spec was %T", j.providerSpec)
	}

	if awsConfig.DiskType.Value == "" {
		awsConfig.DiskType.Value = "standard"
	}

	if awsConfig.DiskSize == 0 {
		awsConfig.DiskSize = 25
	}

	if awsConfig.InstanceType.Value == "" {
		awsConfig.InstanceType.Value = "t3.small"
	}

	if awsConfig.VpcID.Value == "" {
		awsConfig.VpcID.Value = cluster.Spec.Cloud.AWS.VPCID
	}

	if awsConfig.InstanceProfile.Value == "" {
		awsConfig.InstanceProfile.Value = cluster.Spec.Cloud.AWS.InstanceProfileName
	}

	if awsConfig.Region.Value == "" {
		awsConfig.Region.Value = datacenter.Region
	}

	if awsConfig.AvailabilityZone.Value == "" {
		awsConfig.AvailabilityZone.Value = fmt.Sprintf("%sa", awsConfig.Region.Value)
	}

	if len(awsConfig.SecurityGroupIDs) == 0 {
		awsConfig.SecurityGroupIDs = []providerconfig.ConfigVarString{{
			Value: cluster.Spec.Cloud.AWS.SecurityGroupID,
		}}
	}

	awsConfig.Tags = j.clusterTags(cluster)

	return awsConfig, nil
}

func (j *MachineJig) enrichHetznerProviderSpec(datacenter *kubermaticv1.DatacenterSpecHetzner) (interface{}, error) {
	hetznerConfig, ok := j.providerSpec.(hetznertypes.RawConfig)
	if !ok {
		return nil, fmt.Errorf("cluster uses Hetzner, but given provider spec was %T", j.providerSpec)
	}

	if hetznerConfig.Datacenter.Value == "" {
		hetznerConfig.Datacenter.Value = datacenter.Datacenter
	}

	if len(hetznerConfig.Networks) == 0 && datacenter.Network != "" {
		hetznerConfig.Networks = []providerconfig.ConfigVarString{{
			Value: datacenter.Network,
		}}
	}

	if len(hetznerConfig.Networks) == 0 {
		hetznerConfig.Networks = []providerconfig.ConfigVarString{{
			Value: "kubermatic-e2e",
		}}
	}

	return hetznerConfig, nil
}
