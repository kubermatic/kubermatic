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
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/machine"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	nodetypes "k8c.io/machine-controller/sdk/node"
	"k8c.io/machine-controller/sdk/providerconfig"
	"k8c.io/operating-system-manager/pkg/providerconfig/rhel"
	"k8c.io/operating-system-manager/pkg/providerconfig/ubuntu"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type MachineJig struct {
	client ctrlruntimeclient.Client
	log    *zap.SugaredLogger

	// Either specify the cluster directly or specify the clusterJig,
	// if at the time of creating the MachineJig the cluster doesn't exist yet.
	cluster    *kubermaticv1.Cluster
	clusterJig *ClusterJig

	// user-controlled parameters
	name              string
	replicas          int
	osSpec            interface{}
	cloudProviderSpec interface{}
	sshPubKeys        sets.Set[string]
	networkConfig     *providerconfig.NetworkConfig
	clusterClient     ctrlruntimeclient.Client
}

func NewMachineJig(client ctrlruntimeclient.Client, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) *MachineJig {
	return &MachineJig{
		client:     client,
		log:        log,
		cluster:    cluster,
		name:       "e2e-workers",
		osSpec:     ubuntu.Config{},
		replicas:   1,
		sshPubKeys: sets.New[string](),
	}
}

func (j *MachineJig) Clone() *MachineJig {
	return &MachineJig{
		client:            j.client,
		log:               j.log,
		cluster:           j.cluster,
		clusterJig:        j.clusterJig,
		name:              j.name,
		replicas:          j.replicas,
		osSpec:            j.osSpec,
		cloudProviderSpec: j.cloudProviderSpec,
		clusterClient:     j.clusterClient,
		sshPubKeys:        j.sshPubKeys.Clone(),
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

func (j *MachineJig) WithCloudProviderSpec(spec interface{}) *MachineJig {
	j.cloudProviderSpec = spec
	return j
}

func (j *MachineJig) WithCloudProviderSpecPatch(patcher func(cloudProviderSpec interface{}) interface{}) *MachineJig {
	j.cloudProviderSpec = patcher(j.cloudProviderSpec)
	return j
}

func (j *MachineJig) WithOSSpec(spec interface{}) *MachineJig {
	j.osSpec = spec
	return j
}

func (j *MachineJig) WithNetworkConfig(cfg *providerconfig.NetworkConfig) *MachineJig {
	j.networkConfig = cfg
	return j
}

func (j *MachineJig) AddSSHPublicKey(pubKeys ...string) *MachineJig {
	j.sshPubKeys.Insert(pubKeys...).Delete("") // make sure to not add an empty key by accident
	return j
}

func (j *MachineJig) AddSSHKey(key *kubermaticv1.UserSSHKey) *MachineJig {
	return j.AddSSHPublicKey(key.Spec.PublicKey)
}

func (j *MachineJig) WithUbuntu() *MachineJig {
	return j.WithOSSpec(ubuntu.Config{})
}

func (j *MachineJig) WithRHEL() *MachineJig {
	return j.WithOSSpec(rhel.Config{})
}

// If you already have a cluster client, you can set it with WithClusterClient().
// Otherwise the MachineJig will retrieve a proper client itself.
func (j *MachineJig) WithClusterClient(client ctrlruntimeclient.Client) *MachineJig {
	j.clusterClient = client
	return j
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

	_, datacenter, err := Seed(ctx, j.client, datacenterName)
	if err != nil {
		return fmt.Errorf("failed to determine target datacenter: %w", err)
	}

	providerSpec, err := machine.NewBuilder().
		WithCluster(cluster).
		WithDatacenter(datacenter).
		WithOperatingSystemSpec(j.osSpec).
		WithCloudProviderSpec(j.cloudProviderSpec).
		AddSSHPublicKey(j.sshPubKeys.UnsortedList()...).
		BuildProviderSpec()
	if err != nil {
		return fmt.Errorf("failed to create provider spec: %w", err)
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
			Replicas: ptr.To[int32](int32(j.replicas)),
			Template: clusterv1alpha1.MachineTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: clusterv1alpha1.MachineSpec{
					Versions: clusterv1alpha1.MachineVersionInfo{
						Kubelet: cluster.Spec.Version.String(),
					},
					ProviderSpec: *providerSpec,
				},
			},
		},
	}

	clusterClient, err := j.getClusterClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get cluster client: %w", err)
	}

	utilruntime.Must(clusterv1alpha1.AddToScheme(clusterClient.Scheme()))

	err = wait.PollImmediate(ctx, 1*time.Second, 2*time.Minute, func(ctx context.Context) (error, error) {
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
	return wait.PollLog(ctx, j.log, 30*time.Second, 30*time.Minute, func(ctx context.Context) (error, error) {
		nodeList := corev1.NodeList{}
		err := clusterClient.List(ctx, &nodeList)
		if err != nil {
			return fmt.Errorf("failed to list nodes: %w", err), nil
		}

		readyNodeCount := 0
		unready := sets.New[string]()
		for _, node := range nodeList.Items {
			if kubernetes.IsNodeReady(&node) {
				readyNodeCount++
			} else {
				unready.Insert(node.Name)
			}
		}

		if readyNodeCount != j.replicas {
			return fmt.Errorf("%d of %d nodes are ready (unready: %v)", readyNodeCount, j.replicas, sets.List(unready)), nil
		}

		return nil, nil
	})
}

func (j *MachineJig) WaitForReadyPods(ctx context.Context, clusterClient ctrlruntimeclient.Client) error {
	return wait.PollLog(ctx, j.log, 10*time.Second, 15*time.Minute, func(ctx context.Context) (error, error) {
		pods := corev1.PodList{}
		if err := clusterClient.List(ctx, &pods); err != nil {
			return fmt.Errorf("failed to list pods: %w", err), nil
		}

		if len(pods.Items) == 0 {
			return errors.New("no pods found"), nil
		}

		unready := sets.New[string]()
		for _, pod := range pods.Items {
			if !podIsReadyOrCompleted(&pod) {
				unready.Insert(pod.Name)
			}
		}

		if unready.Len() > 0 {
			return fmt.Errorf("not all pods are ready: %v", sets.List(unready)), nil
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
			n.Annotations[nodetypes.SkipEvictionAnnotationKey] = "true"
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
		err = wait.PollLog(ctx, log, 5*time.Second, 10*time.Minute, func(ctx context.Context) (transient error, terminal error) {
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
