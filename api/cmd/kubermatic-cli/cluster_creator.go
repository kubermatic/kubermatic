package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	log "github.com/golang/glog"
	"gopkg.in/yaml.v2"

	corev1 "k8s.io/api/core/v1"
	kuberrrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clusterclientset "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/cluster"
	machineresource "github.com/kubermatic/kubermatic/api/pkg/resources/machine"
)

func newClusterCreator(runOpts Opts) (*clusterCreator, error) {
	kubeconfig, err := clientcmd.
		NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{
				ExplicitPath: runOpts.Kubeconf.String(),
			},
			&clientcmd.ConfigOverrides{
				CurrentContext: runOpts.Context,
			},
		).ClientConfig()
	if err != nil {
		return nil, err
	}

	return &clusterCreator{
		runOpts:          runOpts,
		seedClient:       kubernetes.NewForConfigOrDie(kubeconfig),
		kubermaticClient: kubermaticclientset.NewForConfigOrDie(kubeconfig),
	}, nil
}

type clusterCreator struct {
	runOpts          Opts
	seedClient       *kubernetes.Clientset
	targetClient     *kubernetes.Clientset
	kubermaticClient *kubermaticclientset.Clientset
	clusterName      string
}

func (ctl *clusterCreator) create(ctx context.Context) error {
	var (
		clusterTemplate apiv1.Cluster
		nodeTemplate    apiv1.Node
	)

	if err := unmarshalObj(ctl.runOpts.ClusterPath, &clusterTemplate); err != nil {
		return err
	}

	dcs, err := ctl.getDatacenters()
	if err != nil {
		return err
	}

	dc, found := dcs[clusterTemplate.Spec.Cloud.DatacenterName]
	if !found {
		return fmt.Errorf("datacenter %s not found", clusterTemplate.Spec.Cloud.DatacenterName)
	}

	cloudProviders := cloud.Providers(dcs)

	clusterSpec, err := cluster.Spec(clusterTemplate, cloudProviders)
	if err != nil {
		return err
	}

	crdCluster, err := ctl.createCluster(kubermaticv1.Cluster{Spec: *clusterSpec})
	if err != nil {
		return err
	}
	log.Infof("created cluster named: %s", crdCluster.Name)

	log.Info("waiting for cluster to become healthy")
	err = wait.Poll(
		1*time.Second,
		ctl.runOpts.ClusterTimeout,
		ctl.healthyClusterCond(ctx, crdCluster.Name))
	if err != nil {
		log.Infof("Cluster failed to come up. Health status: %+v", crdCluster.Status.Health.ClusterHealthStatus)
		return err
	}
	log.Info("cluster control plane is up")

	// refresh cluster object
	crdCluster, err = ctl.kubermaticClient.
		KubermaticV1().
		Clusters().
		Get(crdCluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	log.Infof("seed cluster namespace: %s", crdCluster.Status.NamespaceName)
	log.Info("cluster name:")
	fmt.Println(crdCluster.ObjectMeta.Name) // log to STDOUT, for saving in shell

	if err = ctl.installAddons(crdCluster); err != nil {
		return err
	}

	clusterKubeConfig, err := ctl.kubeConfig(crdCluster)
	if err != nil {
		return err
	}

	clusterAdminRestConfig, err := ctl.clusterRestConfig(clusterKubeConfig)
	if err != nil {
		return err
	}

	ctl.targetClient, err = kubernetes.NewForConfig(clusterAdminRestConfig)
	if err != nil {
		return err
	}

	if err = unmarshalObj(ctl.runOpts.NodePath, &nodeTemplate); err != nil {
		return err
	}

	if err = ctl.createMachineDeployment(clusterAdminRestConfig, dc, crdCluster, nodeTemplate); err != nil {
		return err
	}
	log.Info("waiting for machines to boot")

	err = wait.Poll(1*time.Second, ctl.runOpts.NodesTimeout, ctl.nodesReadyCond(ctx))
	if err != nil {
		return err
	}
	log.Info("all nodes are ready")
	log.Infof("target cluster adminconfig: %s", ctl.runOpts.Output)

	return ioutil.WriteFile(ctl.runOpts.Output, clusterKubeConfig, 0600)
}

func (ctl *clusterCreator) delete() error {
	return ctl.kubermaticClient.Kubermatic().Clusters().Delete(ctl.clusterName, nil)
}

func (ctl *clusterCreator) nodesReadyCond(ctx context.Context) func() (bool, error) {
	return func() (bool, error) {
		nodelist, err := ctl.targetClient.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			return false, err
		}

		if len(nodelist.Items) != ctl.runOpts.Nodes {
			// nodes still booting probably
			return false, ctx.Err()
		}

		for _, node := range nodelist.Items {
			for _, cnd := range node.Status.Conditions {
				if cnd.Type == corev1.NodeReady &&
					cnd.Status != corev1.ConditionTrue {
					// Kubelet didn't reported node status "Ready" yet
					return false, ctx.Err()
				}
			}
		}

		return true, ctx.Err()
	}
}

func (ctl *clusterCreator) getDatacenters() (map[string]provider.DatacenterMeta, error) {
	secret, err := ctl.seedClient.
		CoreV1().
		Secrets(ctl.runOpts.KubermaticNamespace).
		Get("datacenters", metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	dcsBuf, found := secret.Data["datacenters.yaml"]
	if !found {
		return nil, errors.New("datacenters.yaml not found")
	}

	dcs := struct {
		Datacenters map[string]provider.DatacenterMeta `yaml:"datacenters"`
	}{}

	if err = yaml.Unmarshal(dcsBuf, &dcs); err != nil {
		return nil, err
	}

	return dcs.Datacenters, nil
}

func (ctl *clusterCreator) createMachineDeployment(restConfig *rest.Config, dc provider.DatacenterMeta, cluster *kubermaticv1.Cluster, node apiv1.Node) error {
	clusterClient, err := clusterclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	machineReplicas := int32(ctl.runOpts.Nodes)
	machineDeploymentClient := clusterClient.ClusterV1alpha1().MachineDeployments(metav1.NamespaceSystem)

	nd := apiv1.NodeDeployment{
		ObjectMeta: apiv1.ObjectMeta{
			Name: node.Name,
		},
		Spec: apiv1.NodeDeploymentSpec{
			Replicas: machineReplicas,
			Template: node.Spec,
		},
	}

	template, err := machineresource.Deployment(cluster, &nd, dc, nil)
	if err != nil {
		return err
	}

	md, err := machineDeploymentClient.Create(template)
	if err != nil {
		return err
	}

	log.Infof("created machine deployment: %s", md.Name)
	return nil
}

func (ctl *clusterCreator) clusterRestConfig(cfg []byte) (*rest.Config, error) {
	clusterClientCfg, err := clientcmd.Load(cfg)
	if err != nil {
		return nil, err
	}

	return clientcmd.NewNonInteractiveClientConfig(
		*clusterClientCfg,
		resources.KubeconfigDefaultContextKey,
		&clientcmd.ConfigOverrides{},
		nil,
	).ClientConfig()
}

func (ctl *clusterCreator) kubeConfig(cluster *kubermaticv1.Cluster) ([]byte, error) {
	secret, err := ctl.seedClient.
		CoreV1().
		Secrets(cluster.Status.NamespaceName).
		Get(resources.AdminKubeconfigSecretName, metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	adminKubeConfig, ok := secret.Data[resources.KubeconfigSecretKey]
	if !ok {
		return nil, fmt.Errorf("%s/%s secret not found", cluster.Status.NamespaceName, resources.KubeconfigSecretKey)
	}

	return adminKubeConfig, nil
}

func (ctl *clusterCreator) createCluster(cluster kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	cluster.GenerateName = cluster.Spec.HumanReadableName
	cluster.Name = ""
	return ctl.kubermaticClient.KubermaticV1().Clusters().Create(&cluster)
}

func (ctl *clusterCreator) healthyClusterCond(ctx context.Context, name string) func() (bool, error) {
	clustersCluent := ctl.kubermaticClient.KubermaticV1().Clusters()

	return func() (bool, error) {
		cluster, err := clustersCluent.Get(name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		return cluster.Status.Health.ClusterHealthStatus.AllHealthy(), ctx.Err()
	}
}

func (ctl *clusterCreator) installAddons(cluster *kubermaticv1.Cluster) error {
	k8cgv := kubermaticv1.SchemeGroupVersion
	ownerRef := *metav1.NewControllerRef(cluster, k8cgv.WithKind("Cluster"))
	addonsClient := ctl.kubermaticClient.KubermaticV1().Addons(cluster.Status.NamespaceName)
	clusterRef := corev1.ObjectReference{
		Name:       cluster.Name,
		Namespace:  "",
		UID:        cluster.UID,
		APIVersion: cluster.APIVersion,
		Kind:       "Cluster",
	}

	for _, addon := range ctl.runOpts.Addons {
		_, err := addonsClient.Create(&kubermaticv1.Addon{
			ObjectMeta: metav1.ObjectMeta{
				Name:            addon,
				Namespace:       cluster.Status.NamespaceName,
				OwnerReferences: []metav1.OwnerReference{ownerRef},
			},
			Spec: kubermaticv1.AddonSpec{
				Name:    addon,
				Cluster: clusterRef,
			},
		})
		if err != nil {
			if kuberrrors.IsAlreadyExists(err) {
				continue
			}
			return err
		}
		log.Infof("created addon: %s", addon)
	}

	return nil
}

func unmarshalObj(fspath string, obj interface{}) error {
	objYaml, err := ioutil.ReadFile(fspath)
	if err != nil {
		return err
	}

	objJSON, err := kubeyaml.ToJSON(objYaml)
	if err != nil {
		return err
	}

	return json.Unmarshal(objJSON, obj)
}
