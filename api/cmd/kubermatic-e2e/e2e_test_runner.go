package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"time"

	log "github.com/golang/glog"
	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	kuberrrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	apiv2 "github.com/kubermatic/kubermatic/api/pkg/api/v2"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	machineresource "github.com/kubermatic/kubermatic/api/pkg/resources/machine"
	machineclientset "github.com/kubermatic/machine-controller/pkg/client/clientset/versioned"
)

const (
	e2eGenerateName   = "e2e-test-runner-"
	ginkgoSerial      = `\[Serial\]`
	ginkgoConformance = `\[Conformance\]`
)

func newE2ETestRunner(runOpts Opts) (*e2eTestRunner, error) {
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", runOpts.KubeconfPath)
	if err != nil {
		return nil, err
	}

	return &e2eTestRunner{
		runOpts:          runOpts,
		seedClient:       kubernetes.NewForConfigOrDie(kubeconfig),
		kubermaticClient: kubermaticclientset.NewForConfigOrDie(kubeconfig),
	}, nil
}

type e2eTestRunner struct {
	runOpts          Opts
	seedClient       *kubernetes.Clientset
	targetClient     *kubernetes.Clientset
	kubermaticClient *kubermaticclientset.Clientset
}

func (ctl *e2eTestRunner) run(ctx context.Context) error {
	var (
		clusterTemplate kubermaticv1.Cluster
		nodeTemplate    apiv2.Node
	)

	if err := unmarshalObj(ctl.runOpts.ClusterPath, &clusterTemplate); err != nil {
		return err
	}

	dc, err := ctl.getDatacenter(clusterTemplate.Spec.Cloud.DatacenterName)
	if err != nil {
		return err
	}

	cluster, err := ctl.createCluster(clusterTemplate)
	if err != nil {
		return err
	}
	if ctl.runOpts.DeleteCluster {
		defer ctl.deleteCluster(cluster.Name)
	}
	log.Infof("created cluster named: %s", cluster.Name)

	log.Info("waiting for cluster to become healthy")
	err = wait.Poll(
		1*time.Second,
		ctl.runOpts.ClusterTimeout,
		ctl.healthyClusterCond(ctx, cluster.Name))
	if err != nil {
		return err
	}
	log.Info("cluster control plane is up")

	// refresh cluster object
	cluster, err = ctl.kubermaticClient.
		KubermaticV1().
		Clusters().
		Get(cluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	log.Infof("seed cluster namespace: %s", cluster.Status.NamespaceName)

	if err = ctl.installAddons(cluster); err != nil {
		return err
	}

	clusterKubeConfig, err := ctl.kubeConfig(cluster)
	if err != nil {
		return err
	}

	clusterAdminRestConfig, err := ctl.clusterRestConfig(clusterKubeConfig, cluster.Name)
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

	if err = ctl.createMachines(clusterAdminRestConfig, dc, cluster, nodeTemplate); err != nil {
		return err
	}
	log.Info("waiting for machines to boot")

	err = wait.Poll(1*time.Second, ctl.runOpts.NodesTimeout, ctl.nodesReadyCond(ctx))
	if err != nil {
		return err
	}
	log.Info("all nodes are ready")

	e2eKubeConfig, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}

	defer func() {
		if err2 := os.Remove(e2eKubeConfig.Name()); err2 != nil {
			log.Error(err2)
		}
	}()
	log.Infof("target cluster adminconfig: %s", e2eKubeConfig.Name())

	if _, err = e2eKubeConfig.Write(clusterKubeConfig); err != nil {
		return err
	}

	if err = e2eKubeConfig.Sync(); err != nil {
		return err
	}

	execCtx, execCancel := context.WithTimeout(ctx, ctl.runOpts.GinkgoTimeout)
	defer execCancel()

	if ctl.runOpts.Parallel == 1 {
		return execGinkgo(execCtx, ctl.runOpts, e2eKubeConfig.Name())
	}

	// copy options for local edits
	parallelOpts := ctl.runOpts

	if parallelOpts.Skip != "" {
		parallelOpts.Skip = fmt.Sprintf("%s|%s", parallelOpts.Skip, ginkgoSerial)
	} else {
		parallelOpts.Skip = ginkgoSerial
	}

	// first run only parallel safe (skip [Serial]) tests
	if err := execGinkgo(execCtx, parallelOpts, e2eKubeConfig.Name()); err != nil {
		return err
	}

	parallelOpts.Parallel = 1
	// restore original skip
	parallelOpts.Skip = ctl.runOpts.Skip

	if parallelOpts.Focus == ginkgoConformance {
		parallelOpts.Focus = fmt.Sprintf("%s.*%s", ginkgoSerial, ginkgoConformance)
	} else {
		parallelOpts.Focus = fmt.Sprintf("%s.*%s.*%s", parallelOpts.Focus, ginkgoSerial, ginkgoConformance)
	}

	// second run only [Serial] conformance tests
	return execGinkgo(execCtx, parallelOpts, e2eKubeConfig.Name())
}

func execGinkgo(ctx context.Context, opts Opts, kubeconfig string) error {
	cmd := exec.CommandContext(ctx,
		opts.GinkgoBin,
		`--focus=`+opts.Focus,
		"--skip="+opts.Skip,
		"--noColor="+strconv.FormatBool(opts.GinkgoNoColor),
		"--nodes="+strconv.Itoa(opts.Parallel),
		opts.TestBin,
		"--",
		"--disable-log-dump",
		"--repo-root=/kubernetes",
		"--provider="+opts.Provider,
		"--report-dir="+opts.ReportsDir,
		"--kubeconfig="+kubeconfig)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Infof("running command: %v", cmd.Args)

	return cmd.Run()
}

func (ctl *e2eTestRunner) nodesReadyCond(ctx context.Context) func() (bool, error) {
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

func (ctl *e2eTestRunner) getDatacenter(name string) (provider.DatacenterMeta, error) {
	secret, err := ctl.seedClient.
		CoreV1().
		Secrets(ctl.runOpts.KubermaticNamespace).
		Get("datacenters", metav1.GetOptions{})

	if err != nil {
		return provider.DatacenterMeta{}, err
	}

	dcsBuf, found := secret.Data["datacenters.yaml"]
	if !found {
		return provider.DatacenterMeta{}, errors.New("datacenters.yaml not found")
	}

	dcs := struct {
		Datacenters map[string]provider.DatacenterMeta `yaml:"datacenters"`
	}{}

	if err = yaml.Unmarshal(dcsBuf, &dcs); err != nil {
		return provider.DatacenterMeta{}, err
	}

	dc, found := dcs.Datacenters[name]
	if !found {
		return provider.DatacenterMeta{}, fmt.Errorf("datacenter %s not found", name)
	}

	return dc, nil
}

func (ctl *e2eTestRunner) createMachines(restConfig *rest.Config, dc provider.DatacenterMeta, cluster *kubermaticv1.Cluster, node apiv2.Node) error {
	machinesClient, err := machineclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	machines := machinesClient.MachineV1alpha1().Machines()
	template, err := machineresource.Machine(cluster, &node, dc, nil)
	if err != nil {
		return err
	}

	for i := 0; i < ctl.runOpts.Nodes; i++ {
		template.Name = fmt.Sprintf("%s%s", e2eGenerateName, rand.String(5))
		template.Spec.Name = template.Name

		m, err := machines.Create(template)
		if err != nil {
			return err
		}
		log.Infof("created machine: %s", m.Name)
	}

	return nil
}

func (ctl *e2eTestRunner) clusterRestConfig(cfg []byte, contextName string) (*rest.Config, error) {
	clusterClientCfg, err := clientcmd.Load(cfg)
	if err != nil {
		return nil, err
	}

	return clientcmd.NewNonInteractiveClientConfig(
		*clusterClientCfg,
		contextName,
		&clientcmd.ConfigOverrides{},
		nil,
	).ClientConfig()
}

func (ctl *e2eTestRunner) kubeConfig(cluster *kubermaticv1.Cluster) ([]byte, error) {
	secret, err := ctl.seedClient.
		CoreV1().
		Secrets(cluster.Status.NamespaceName).
		Get(resources.AdminKubeconfigSecretName, metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	adminKubeConfig, ok := secret.Data[resources.AdminKubeconfigSecretKey]
	if !ok {
		return nil, errors.New("admin-kubeconfig not found")
	}

	return adminKubeConfig, nil
}

func (ctl *e2eTestRunner) createCluster(cluster kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	cluster.GenerateName = e2eGenerateName
	cluster.Name = ""
	return ctl.kubermaticClient.KubermaticV1().Clusters().Create(&cluster)
}

func (ctl *e2eTestRunner) deleteCluster(name string) {
	log.Infof("deleting cluster: %s", name)
	err := ctl.kubermaticClient.KubermaticV1().Clusters().Delete(name, nil)
	if err != nil {
		log.Error(err)
	}
}

func (ctl *e2eTestRunner) healthyClusterCond(ctx context.Context, name string) func() (bool, error) {
	clustersCluent := ctl.kubermaticClient.KubermaticV1().Clusters()

	return func() (bool, error) {
		cluster, err := clustersCluent.Get(name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		return cluster.Status.Health.ClusterHealthStatus.AllHealthy(), ctx.Err()
	}
}

func (ctl *e2eTestRunner) installAddons(cluster *kubermaticv1.Cluster) error {
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
