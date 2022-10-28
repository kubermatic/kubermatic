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
	alibabatypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/alibaba/types"
	awstypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/aws/types"
	azuretypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/azure/types"
	digitaloceantypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/digitalocean/types"
	equinixmetaltypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/equinixmetal/types"
	gcptypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/gce/types"
	hetznertypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/hetzner/types"
	openstacktypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/openstack/types"
	vspheretypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vsphere/types"
	evictiontypes "github.com/kubermatic/machine-controller/pkg/node/eviction/types"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	"k8c.io/operating-system-manager/pkg/providerconfig/amzn2"
	"k8c.io/operating-system-manager/pkg/providerconfig/centos"
	"k8c.io/operating-system-manager/pkg/providerconfig/flatcar"
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
	"k8s.io/client-go/util/retry"
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
	networkConfig *providerconfig.NetworkConfig
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

func (j *MachineJig) Clone() *MachineJig {
	return &MachineJig{
		client:        j.client,
		log:           j.log,
		cluster:       j.cluster,
		clusterJig:    j.clusterJig,
		name:          j.name,
		replicas:      j.replicas,
		osSpec:        j.osSpec,
		providerSpec:  j.providerSpec,
		clusterClient: j.clusterClient,
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

func (j *MachineJig) WithNetworkConfig(cfg *providerconfig.NetworkConfig) *MachineJig {
	j.networkConfig = cfg
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

func (j *MachineJig) WithProviderPatch(patcher func(providerSpec interface{}) interface{}) *MachineJig {
	j.providerSpec = patcher(j.providerSpec)
	return j
}

// If you already have a cluster client, you can set it with WithClusterClient().
// Otherwise the MachineJig will retrieve a proper client itself.
func (j *MachineJig) WithClusterClient(client ctrlruntimeclient.Client) *MachineJig {
	j.clusterClient = client
	return j
}

func (j *MachineJig) WithAlibaba(instanceType string, diskSizeGB int) *MachineJig {
	return j.WithProviderSpec(alibabatypes.RawConfig{
		InstanceType:            providerconfig.ConfigVarString{Value: instanceType},
		DiskSize:                providerconfig.ConfigVarString{Value: fmt.Sprintf("%d", diskSizeGB)},
		DiskType:                providerconfig.ConfigVarString{Value: "cloud"},
		InternetMaxBandwidthOut: providerconfig.ConfigVarString{Value: "10"},
	})
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

func (j *MachineJig) WithAzure(vmSize string) *MachineJig {
	return j.WithProviderSpec(azuretypes.RawConfig{
		VMSize: providerconfig.ConfigVarString{Value: vmSize},
	})
}

func (j *MachineJig) WithHetzner(instanceSize string) *MachineJig {
	return j.WithProviderSpec(hetznertypes.RawConfig{
		ServerType: providerconfig.ConfigVarString{Value: instanceSize},
	})
}

func (j *MachineJig) WithOpenstack(flavor string) *MachineJig {
	return j.WithProviderSpec(openstacktypes.RawConfig{
		Flavor: providerconfig.ConfigVarString{Value: flavor},
	})
}

func (j *MachineJig) WithVSphere(cpus int, memory int, diskSizeGB int) *MachineJig {
	return j.WithProviderSpec(vspheretypes.RawConfig{
		CPUs:       int32(cpus),
		MemoryMB:   int64(memory),
		DiskSizeGB: pointer.Int64(int64(diskSizeGB)),
	})
}

func (j *MachineJig) WithGCP(machineType string, diskSize int, preemtible bool) *MachineJig {
	return j.WithProviderSpec(gcptypes.RawConfig{
		MachineType: providerconfig.ConfigVarString{Value: machineType},
		DiskSize:    int64(diskSize),
		DiskType:    providerconfig.ConfigVarString{Value: "pd-standard"},
		Preemptible: providerconfig.ConfigVarBool{Value: &preemtible},
	})
}

func (j *MachineJig) WithDigitalocean(size string) *MachineJig {
	return j.WithProviderSpec(digitaloceantypes.RawConfig{
		Size:       providerconfig.ConfigVarString{Value: size},
		Backups:    providerconfig.ConfigVarBool{Value: pointer.Bool(false)},
		Monitoring: providerconfig.ConfigVarBool{Value: pointer.Bool(false)},
	})
}

func (j *MachineJig) WithEquinixMetal(instanceType string) *MachineJig {
	return j.WithProviderSpec(equinixmetaltypes.RawConfig{
		InstanceType: providerconfig.ConfigVarString{Value: instanceType},
	})
}

type MachineWaitMode string

const (
	WaitForNothing    MachineWaitMode = ""
	WaitForReadyNodes MachineWaitMode = "nodes"
	WaitForReadyPods  MachineWaitMode = "pods"
)

func (j *MachineJig) Create(ctx context.Context, waitMode MachineWaitMode, datacenterName string) error {
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

	_, datacenter, err := Seed(ctx, j.client, datacenterName)
	if err != nil {
		return fmt.Errorf("failed to determine target datacenter: %w", err)
	}

	providerSpec, err := j.enrichProviderSpec(cluster, provider, datacenter, os)
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
		Network: j.networkConfig,
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
		if err = j.WaitForReadyNodes(ctx, clusterClient); err != nil {
			return fmt.Errorf("failed to wait: %w", err)
		}
	}

	if waitMode == WaitForReadyPods {
		j.log.Info("Waiting for pods to become ready...")
		if err = j.WaitForReadyPods(ctx, clusterClient); err != nil {
			return fmt.Errorf("failed to wait: %w", err)
		}
	}

	return nil
}

func (j *MachineJig) WaitForReadyNodes(ctx context.Context, clusterClient ctrlruntimeclient.Client) error {
	return wait.PollLog(ctx, j.log, 30*time.Second, 30*time.Minute, func() (error, error) {
		nodeList := corev1.NodeList{}
		err := clusterClient.List(ctx, &nodeList)
		if err != nil {
			return fmt.Errorf("failed to list nodes: %w", err), nil
		}

		readyNodeCount := 0
		unready := sets.NewString()
		for _, node := range nodeList.Items {
			if kubernetes.IsNodeReady(&node) {
				readyNodeCount++
			} else {
				unready.Insert(node.Name)
			}
		}

		if readyNodeCount != j.replicas {
			return fmt.Errorf("%d of %d nodes are ready (unready: %v)", readyNodeCount, j.replicas, unready.List()), nil
		}

		return nil, nil
	})
}

func (j *MachineJig) WaitForReadyPods(ctx context.Context, clusterClient ctrlruntimeclient.Client) error {
	return wait.PollLog(ctx, j.log, 10*time.Second, 15*time.Minute, func() (error, error) {
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

func (j *MachineJig) SkipEvictionForAllNodes(ctx context.Context, clusterClient ctrlruntimeclient.Client) error {
	nodes := &corev1.NodeList{}
	if err := clusterClient.List(ctx, nodes); err != nil {
		return fmt.Errorf("failed to list user cluster nodes: %w", err)
	}

	for _, node := range nodes.Items {
		nodeKey := ctrlruntimeclient.ObjectKey{Name: node.Name}
		j.log.Debugw("Marking node with skip-eviction...", "node", node.Name)

		retErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			n := corev1.Node{}
			if err := clusterClient.Get(ctx, nodeKey, &n); err != nil {
				return err
			}

			if n.Annotations == nil {
				n.Annotations = map[string]string{}
			}
			n.Annotations[evictiontypes.SkipEvictionAnnotationKey] = "true"
			return clusterClient.Update(ctx, &n)
		})
		if retErr != nil {
			return fmt.Errorf("failed to annotate node %s: %w", node.Name, retErr)
		}
	}

	return nil
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

	clusterJig := j.clusterJig
	if clusterJig == nil {
		clusterJig = NewClusterJig(j.client, j.log)
	}

	return clusterJig.
		WithExistingCluster(cluster.Name).
		WithProjectName(projectName).
		ClusterClient(ctx)
}

// determineCloudProvider determines the machine-controller (!) provider type
// based on the given cluster. Note that the MC potentially uses different
// provider names than KKP ("gcp" vs. "gce" for example).
func (j *MachineJig) determineCloudProvider(cluster *kubermaticv1.Cluster) (providerconfig.CloudProvider, error) {
	name := cluster.Spec.Cloud.ProviderName

	// machine-controller uses "gce" and names it "google", KKP calls it consistently "gcp"
	if name == string(kubermaticv1.GCPCloudProvider) {
		name = string(providerconfig.CloudProviderGoogle)
	}

	provider := providerconfig.CloudProvider(name)

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
	case flatcar.Config:
		return providerconfig.OperatingSystemFlatcar, nil
	}

	return "", errors.New("cannot determine OS from the given osSpec")
}

// enrichProviderSpec takes the providerSpec (i.e. the machine config from the testcase, usually set
// by one of the preset functions, for AWS this might be instance type + disk size) and fills in the
// other required fields (for AWS for example the VPCID or instance profile name) based on the datacenter
// (static configuration) and the cluster object (dynamic infos that some providers write into the spec).
// The result is the providerSpec being ready to be marshalled into a MachineSpec to ultimately create
// the MachineDeployment.
func (j *MachineJig) enrichProviderSpec(cluster *kubermaticv1.Cluster, provider providerconfig.CloudProvider, datacenter *kubermaticv1.Datacenter, os providerconfig.OperatingSystem) (interface{}, error) {
	switch provider {
	case providerconfig.CloudProviderAlibaba:
		return j.enrichAlibabaProviderSpec(cluster, datacenter.Spec.Alibaba)
	case providerconfig.CloudProviderAWS:
		return j.enrichAWSProviderSpec(cluster, datacenter.Spec.AWS)
	case providerconfig.CloudProviderAzure:
		return j.enrichAzureProviderSpec(cluster, datacenter.Spec.Azure)
	case providerconfig.CloudProviderHetzner:
		return j.enrichHetznerProviderSpec(datacenter.Spec.Hetzner)
	case providerconfig.CloudProviderOpenstack:
		return j.enrichOpenstackProviderSpec(cluster, datacenter.Spec.Openstack, os)
	case providerconfig.CloudProviderVsphere:
		return j.enrichVSphereProviderSpec(cluster, datacenter.Spec.VSphere, os)
	case providerconfig.CloudProviderGoogle:
		return j.enrichGCPProviderSpec(cluster, datacenter.Spec.GCP, os)
	case providerconfig.CloudProviderDigitalocean:
		return j.enrichDigitaloceanProviderSpec(cluster, datacenter.Spec.Digitalocean, os)
	case providerconfig.CloudProviderPacket:
		return j.enrichEquinixMetalProviderSpec(cluster, datacenter.Spec.Packet, os)
	default:
		return nil, fmt.Errorf("don't know how to handle %q provider specs", provider)
	}
}

func (j *MachineJig) enrichAlibabaProviderSpec(cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecAlibaba) (interface{}, error) {
	alibabaConfig, ok := j.providerSpec.(alibabatypes.RawConfig)
	if !ok {
		return nil, fmt.Errorf("cluster uses Alibaba, but given provider spec was %T", j.providerSpec)
	}

	if alibabaConfig.DiskType.Value == "" {
		alibabaConfig.DiskType.Value = "cloud"
	}

	if alibabaConfig.DiskSize.Value == "" {
		alibabaConfig.DiskSize.Value = "40"
	}

	if alibabaConfig.InternetMaxBandwidthOut.Value == "" {
		alibabaConfig.InternetMaxBandwidthOut.Value = "10"
	}

	if alibabaConfig.RegionID.Value == "" {
		alibabaConfig.RegionID.Value = datacenter.Region
	}

	if alibabaConfig.ZoneID.Value == "" {
		alibabaConfig.ZoneID.Value = fmt.Sprintf("%sa", alibabaConfig.RegionID.Value)
	}

	return alibabaConfig, nil
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

	awsConfig.Tags = map[string]string{
		"kubernetes.io/cluster/" + cluster.Name: "",
		"system/cluster":                        cluster.Name,
	}

	if projectID, ok := cluster.Labels[kubermaticv1.ProjectIDLabelKey]; ok {
		awsConfig.Tags["system/project"] = projectID
	}

	return awsConfig, nil
}

func (j *MachineJig) enrichAzureProviderSpec(cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecAzure) (interface{}, error) {
	azureConfig, ok := j.providerSpec.(azuretypes.RawConfig)
	if !ok {
		return nil, fmt.Errorf("cluster uses Azure, but given provider spec was %T", j.providerSpec)
	}

	if azureConfig.AssignAvailabilitySet == nil {
		azureConfig.AssignAvailabilitySet = cluster.Spec.Cloud.Azure.AssignAvailabilitySet
	}

	if azureConfig.AvailabilitySet.Value == "" {
		azureConfig.AvailabilitySet.Value = cluster.Spec.Cloud.Azure.AvailabilitySet
	}

	if azureConfig.Location.Value == "" {
		azureConfig.Location.Value = datacenter.Location
	}

	if azureConfig.ResourceGroup.Value == "" {
		azureConfig.ResourceGroup.Value = cluster.Spec.Cloud.Azure.ResourceGroup
	}

	if azureConfig.VNetResourceGroup.Value == "" {
		azureConfig.VNetResourceGroup.Value = cluster.Spec.Cloud.Azure.VNetResourceGroup
	}

	if azureConfig.VNetName.Value == "" {
		azureConfig.VNetName.Value = cluster.Spec.Cloud.Azure.VNetName
	}

	if azureConfig.SubnetName.Value == "" {
		azureConfig.SubnetName.Value = cluster.Spec.Cloud.Azure.SubnetName
	}

	if azureConfig.RouteTableName.Value == "" {
		azureConfig.RouteTableName.Value = cluster.Spec.Cloud.Azure.RouteTableName
	}

	if azureConfig.SecurityGroupName.Value == "" {
		azureConfig.SecurityGroupName.Value = cluster.Spec.Cloud.Azure.SecurityGroup
	}

	if azureConfig.LoadBalancerSku.Value == "" {
		azureConfig.LoadBalancerSku.Value = string(cluster.Spec.Cloud.Azure.LoadBalancerSKU)
	}

	azureConfig.Tags = map[string]string{
		"KubernetesCluster": cluster.Name,
		"system-cluster":    cluster.Name,
	}

	if projectID, ok := cluster.Labels[kubermaticv1.ProjectIDLabelKey]; ok {
		azureConfig.Tags["system-project"] = projectID
	}

	return azureConfig, nil
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

func (j *MachineJig) enrichOpenstackProviderSpec(cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecOpenstack, os providerconfig.OperatingSystem) (interface{}, error) {
	openstackConfig, ok := j.providerSpec.(openstacktypes.RawConfig)
	if !ok {
		return nil, fmt.Errorf("cluster uses Openstack, but given provider spec was %T", j.providerSpec)
	}

	image, ok := datacenter.Images[os]
	if !ok {
		return nil, fmt.Errorf("no disk image configured for operating system %q", os)
	}

	openstackConfig.Image.Value = image

	if openstackConfig.AvailabilityZone.Value == "" {
		openstackConfig.AvailabilityZone.Value = datacenter.AvailabilityZone
	}

	if openstackConfig.Region.Value == "" {
		openstackConfig.Region.Value = datacenter.Region
	}

	if openstackConfig.IdentityEndpoint.Value == "" {
		openstackConfig.IdentityEndpoint.Value = datacenter.AuthURL
	}

	if openstackConfig.FloatingIPPool.Value == "" {
		openstackConfig.FloatingIPPool.Value = cluster.Spec.Cloud.Openstack.FloatingIPPool
	}

	if openstackConfig.Network.Value == "" {
		openstackConfig.Network.Value = cluster.Spec.Cloud.Openstack.Network
	}

	if openstackConfig.Subnet.Value == "" {
		openstackConfig.Subnet.Value = cluster.Spec.Cloud.Openstack.SubnetID
	}

	if len(openstackConfig.SecurityGroups) == 0 {
		openstackConfig.SecurityGroups = []providerconfig.ConfigVarString{{Value: cluster.Spec.Cloud.Openstack.SecurityGroups}}
	}

	if openstackConfig.TrustDevicePath.Value == nil {
		openstackConfig.TrustDevicePath.Value = datacenter.TrustDevicePath
	}

	openstackConfig.Tags = map[string]string{
		"KubernetesCluster": cluster.Name,
		"system-cluster":    cluster.Name,
	}

	if projectID, ok := cluster.Labels[kubermaticv1.ProjectIDLabelKey]; ok {
		openstackConfig.Tags["system-project"] = projectID
	}

	return openstackConfig, nil
}

func (j *MachineJig) enrichVSphereProviderSpec(cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecVSphere, os providerconfig.OperatingSystem) (interface{}, error) {
	vsphereConfig, ok := j.providerSpec.(vspheretypes.RawConfig)
	if !ok {
		return nil, fmt.Errorf("cluster uses VSphere, but given provider spec was %T", j.providerSpec)
	}

	template, ok := datacenter.Templates[os]
	if !ok {
		return nil, fmt.Errorf("no VM template configured for operating system %q", os)
	}

	vsphereConfig.TemplateVMName.Value = template

	var datastore = ""
	// If `DatastoreCluster` is not specified we use either the Datastore
	// specified at `Cluster` or the one specified at `Datacenter` level.
	if cluster.Spec.Cloud.VSphere.DatastoreCluster == "" {
		datastore = cluster.Spec.Cloud.VSphere.Datastore
		if datastore == "" {
			datastore = datacenter.DefaultDatastore
		}
	}

	if vsphereConfig.Datastore.Value == "" {
		vsphereConfig.Datastore.Value = datastore
	}

	if vsphereConfig.Folder.Value == "" {
		vsphereConfig.Folder.Value = fmt.Sprintf("%s/%s", datacenter.RootPath, cluster.Name)
	}

	if vsphereConfig.Datacenter.Value == "" {
		vsphereConfig.Datacenter.Value = datacenter.Datacenter
	}

	if vsphereConfig.Cluster.Value == "" {
		vsphereConfig.Cluster.Value = datacenter.Cluster
	}

	if vsphereConfig.AllowInsecure.Value == nil {
		vsphereConfig.AllowInsecure.Value = pointer.Bool(datacenter.AllowInsecure)
	}

	if vsphereConfig.VMNetName.Value == "" {
		vsphereConfig.VMNetName.Value = cluster.Spec.Cloud.VSphere.VMNetName
	}

	if vsphereConfig.DatastoreCluster.Value == "" {
		vsphereConfig.DatastoreCluster.Value = cluster.Spec.Cloud.VSphere.DatastoreCluster
	}

	if vsphereConfig.Folder.Value == "" {
		vsphereConfig.Folder.Value = cluster.Spec.Cloud.VSphere.Folder
	}

	if vsphereConfig.ResourcePool.Value == "" {
		vsphereConfig.ResourcePool.Value = cluster.Spec.Cloud.VSphere.ResourcePool
	}

	return vsphereConfig, nil
}

func (j *MachineJig) enrichGCPProviderSpec(cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecGCP, os providerconfig.OperatingSystem) (interface{}, error) {
	gcpConfig, ok := j.providerSpec.(gcptypes.RawConfig)
	if !ok {
		return nil, fmt.Errorf("cluster uses GCP, but given provider spec was %T", j.providerSpec)
	}

	if gcpConfig.Regional.Value == nil {
		gcpConfig.Regional.Value = &datacenter.Regional
	}

	if gcpConfig.Zone.Value == "" {
		gcpConfig.Zone.Value = datacenter.Region + "-" + datacenter.ZoneSuffixes[0]
	}

	if gcpConfig.Network.Value == "" {
		gcpConfig.Network.Value = cluster.Spec.Cloud.GCP.Network
	}

	if gcpConfig.Subnetwork.Value == "" {
		gcpConfig.Subnetwork.Value = cluster.Spec.Cloud.GCP.Subnetwork
	}

	gcpConfig.Tags = []string{
		fmt.Sprintf("kubernetes-cluster-%s", cluster.Name),
		fmt.Sprintf("system-cluster-%s", cluster.Name),
	}

	if projectID, ok := cluster.Labels[kubermaticv1.ProjectIDLabelKey]; ok {
		gcpConfig.Tags = append(gcpConfig.Tags, fmt.Sprintf("system-project-%s", projectID))
	}

	return gcpConfig, nil
}

func (j *MachineJig) enrichDigitaloceanProviderSpec(cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecDigitalocean, os providerconfig.OperatingSystem) (interface{}, error) {
	doConfig, ok := j.providerSpec.(digitaloceantypes.RawConfig)
	if !ok {
		return nil, fmt.Errorf("cluster uses Digitalocean, but given provider spec was %T", j.providerSpec)
	}

	if doConfig.Region.Value == "" {
		doConfig.Region.Value = datacenter.Region
	}

	tags := []string{
		"kubernetes",
		fmt.Sprintf("kubernetes-cluster-%s", cluster.Name),
		fmt.Sprintf("system-cluster-%s", cluster.Name),
	}

	if projectID, ok := cluster.Labels[kubermaticv1.ProjectIDLabelKey]; ok {
		tags = append(tags, fmt.Sprintf("system-project-%s", projectID))
	}

	for _, tag := range tags {
		doConfig.Tags = append(doConfig.Tags, providerconfig.ConfigVarString{Value: tag})
	}

	return doConfig, nil
}

func (j *MachineJig) enrichEquinixMetalProviderSpec(cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecPacket, os providerconfig.OperatingSystem) (interface{}, error) {
	emConfig, ok := j.providerSpec.(equinixmetaltypes.RawConfig)
	if !ok {
		return nil, fmt.Errorf("cluster uses Equinix Metal (Packet), but given provider spec was %T", j.providerSpec)
	}

	if emConfig.Metro.Value == "" {
		emConfig.Metro.Value = datacenter.Metro
	}

	if len(emConfig.Facilities) == 0 {
		for _, facility := range datacenter.Facilities {
			emConfig.Facilities = append(emConfig.Facilities, providerconfig.ConfigVarString{Value: facility})
		}
	}

	tags := []string{
		"kubernetes",
		fmt.Sprintf("kubernetes-cluster-%s", cluster.Name),
		fmt.Sprintf("system/cluster:%s", cluster.Name),
	}

	if projectID, ok := cluster.Labels[kubermaticv1.ProjectIDLabelKey]; ok {
		tags = append(tags, fmt.Sprintf("system/project:%s", projectID))
	}

	for _, tag := range tags {
		emConfig.Tags = append(emConfig.Tags, providerconfig.ConfigVarString{Value: tag})
	}

	return emConfig, nil
}
