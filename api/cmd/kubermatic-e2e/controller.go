package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"

	corev1 "k8s.io/api/core/v1"
	kuberrrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	machineclientset "github.com/kubermatic/machine-controller/pkg/client/clientset/versioned"
	machinesv1alpha1 "github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
)

const (
	e2eGenerateName = "e2e-test-runner-"
)

func newE2ETestsController(runOpts Opts) (*e2eTestsController, error) {
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", runOpts.KubeconfPath)
	if err != nil {
		return nil, err
	}

	return &e2eTestsController{
		runOpts:          runOpts,
		seedClient:       kubernetes.NewForConfigOrDie(kubeconfig),
		kubermaticClient: kubermaticclientset.NewForConfigOrDie(kubeconfig),
	}, nil
}

type e2eTestsController struct {
	runOpts          Opts
	seedClient       *kubernetes.Clientset
	targetClient     *kubernetes.Clientset
	kubermaticClient *kubermaticclientset.Clientset
}

func (ctl *e2eTestsController) run(ctx context.Context) error {
	clusterTemplate := &kubermaticv1.Cluster{}
	if err := unmarshalObj(ctl.runOpts.ClusterPath, clusterTemplate); err != nil {
		return err
	}
	clusterTemplate.ObjectMeta.GenerateName = e2eGenerateName
	clusterTemplate.ObjectMeta.Name = ""

	cluster, err := ctl.createCluster(clusterTemplate)
	if err != nil {
		return err
	}
	defer ctl.deleteCluster(cluster.ObjectMeta.Name)
	log.Print("created cluster")

	log.Print("waiting for cluster to became healty")
	err = wait.Poll(
		1*time.Second,
		ctl.runOpts.ClusterTimeout,
		ctl.healthyClusterCond(ctx, cluster.ObjectMeta.Name))
	if err != nil {
		return err
	}
	log.Print("cluster control plain is up")

	// refrash cluster object
	cluster, err = ctl.kubermaticClient.
		KubermaticV1().
		Clusters().
		Get(cluster.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if err = ctl.installAddons(cluster); err != nil {
		return err
	}

	clusterKubeConfig, err := ctl.kubeConfig(cluster)
	if err != nil {
		return err
	}

	clusterAdminRestConfig, err := ctl.clusterRestConfig(clusterKubeConfig, cluster.ObjectMeta.Name)
	if err != nil {
		return err
	}

	ctl.targetClient, err = kubernetes.NewForConfig(clusterAdminRestConfig)
	if err != nil {
		return err
	}

	machineTemplate := &machinesv1alpha1.Machine{}
	if err = unmarshalObj(ctl.runOpts.MachinePath, machineTemplate); err != nil {
		return err
	}
	machineTemplate.ObjectMeta.GenerateName = e2eGenerateName
	machineTemplate.ObjectMeta.Name = ""

	if err = ctl.createMachines(clusterAdminRestConfig, machineTemplate); err != nil {
		return err
	}
	log.Print("waiting for machines to boot")

	err = wait.Poll(1*time.Second, ctl.runOpts.NodesTimeout, ctl.nodesReadyCond(ctx))
	if err != nil {
		return err
	}
	log.Print("all nodes are ready")

	e2eKubeConfig, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}

	defer func() {
		if err2 := os.Remove(e2eKubeConfig.Name()); err2 != nil {
			log.Print(err2)
		}
	}()
	log.Printf("target cluster adminconfig: %s", e2eKubeConfig.Name())

	if _, err = e2eKubeConfig.Write(clusterKubeConfig); err != nil {
		return err
	}

	if err = e2eKubeConfig.Sync(); err != nil {
		return err
	}

	return ctl.execGinkgo(ctx, e2eKubeConfig.Name())
}

func (ctl *e2eTestsController) execGinkgo(ctx context.Context, kubeconfig string) error {
	execCtx, execCancel := context.WithTimeout(ctx, 3600*time.Second)
	defer execCancel()

	cmd := exec.CommandContext(execCtx,
		ctl.runOpts.GinkgoBin,
		"--focus="+ctl.runOpts.Focus,
		"--skip="+ctl.runOpts.Skip,
		"--noColor=true",
		"--nodes="+ctl.runOpts.Parallel,
		ctl.runOpts.TestBin,
		"--",
		"--disable-log-dump",
		"--repo-root=/kubernetes",
		"--provider="+ctl.runOpts.Provider,
		"--report-dir="+ctl.runOpts.ReportsDir,
		"--kubeconfig="+kubeconfig)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("running command: %v", cmd.Args)

	return cmd.Run()
}

func (ctl *e2eTestsController) nodesReadyCond(ctx context.Context) func() (bool, error) {
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

func (ctl *e2eTestsController) createMachines(restConfig *rest.Config, template *machinesv1alpha1.Machine) error {
	machinesClient, err := machineclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	machines := machinesClient.MachineV1alpha1().Machines()
	machineName := template.Spec.ObjectMeta.Name

	for i := 0; i < ctl.runOpts.Nodes; i++ {
		template.Spec.ObjectMeta.Name = fmt.Sprintf("%s-%d", machineName, i)
		if _, err := machines.Create(template); err != nil {
			return err
		}
	}

	return nil
}

func (ctl *e2eTestsController) clusterRestConfig(cfg []byte, contextName string) (*rest.Config, error) {
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

func (ctl *e2eTestsController) kubeConfig(cluster *kubermaticv1.Cluster) ([]byte, error) {
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

func (ctl *e2eTestsController) createCluster(cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	return ctl.kubermaticClient.KubermaticV1().Clusters().Create(cluster)
}

func (ctl *e2eTestsController) deleteCluster(name string) {
	ctl.kubermaticClient.KubermaticV1().Clusters().Delete(name, nil)
}

func (ctl *e2eTestsController) healthyClusterCond(ctx context.Context, name string) func() (bool, error) {
	clustersCluent := ctl.kubermaticClient.KubermaticV1().Clusters()

	return func() (bool, error) {
		cluster, err := clustersCluent.Get(name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		return cluster.Status.Health.ClusterHealthStatus.AllHealthy(), ctx.Err()
	}
}

func (ctl *e2eTestsController) installAddons(cluster *kubermaticv1.Cluster) error {
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
