package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/davecgh/go-spew/spew"
	flag "github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	kuberrrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	k8cclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	k8cv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	k8csignals "github.com/kubermatic/kubermatic/api/pkg/signals"
	machineclientset "github.com/kubermatic/machine-controller/pkg/client/clientset/versioned"
	machinesv1alpha1 "github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
)

// Opts represent combination of flags and ENV options
type Opts struct {
	Addons         []string
	ClusterPath    string
	Focus          string
	GinkgoBin      string
	KubeconfPath   string
	MachinePath    string
	Nodes          int
	Parallel       string
	Provider       string
	ReportsDir     string
	Skip           string
	TestBin        string
	ClusterTimeout time.Duration
	NodesTimeout   time.Duration
}

func main() {
	log.Print("starting")

	var runOpts Opts

	flag.IntVar(&runOpts.Nodes,
		"nodes", 1, "number of worker nodes")
	flag.StringArrayVar(&runOpts.Addons,
		"addons",
		[]string{"canal", "dns", "kube-proxy", "openvpn", "rbac"},
		"coma separated list of addons")
	flag.StringVar(&runOpts.ClusterPath,
		"cluster", "/manifests/cluster.yaml", "path to Cluster yaml")
	flag.StringVar(&runOpts.KubeconfPath,
		"kubeconfig", "/config/kubeconfig", "path to kubeconfig file")
	flag.StringVar(&runOpts.MachinePath,
		"machine", "/manifests/machine.yaml", "path to Machine yaml")
	flag.DurationVar(&runOpts.ClusterTimeout,
		"cluster-timeout", 3*time.Minute, "cluster creation timeout")
	flag.DurationVar(&runOpts.NodesTimeout,
		"nodes-timeout", 10*time.Minute, "nodes creation timeout")

	runOpts.GinkgoBin = lookupEnv("E2E_GINKGO", "/usr/local/bin/ginkgo")
	runOpts.TestBin = lookupEnv("E2E_TEST", "/usr/local/bin/e2e.test")
	runOpts.Focus = lookupEnv("E2E_FOCUS", "[Conformance]")
	runOpts.Skip = lookupEnv("E2E_SKIP", "Alpha|Kubectl|[(Disruptive|Feature:[^]]+|Flaky)]")
	runOpts.Provider = lookupEnv("E2E_PROVIDER", "local")
	runOpts.Parallel = lookupEnv("E2E_PARALLEL", "1")
	runOpts.ReportsDir = lookupEnv("E2E_REPORTS_DIR", "/tmp/results")
	flag.Parse()

	spew.Dump(runOpts)

	stopCh := k8csignals.SetupSignalHandler()
	rootCtx, rootCancel := context.WithCancel(context.Background())

	go func() {
		select {
		case <-stopCh:
			rootCancel()
			log.Print("user requested to stop the application")
		case <-rootCtx.Done():
			log.Print("parent context has been closed - propagating the request")
		}
	}()

	kubeconfig, err := clientcmd.BuildConfigFromFlags("", runOpts.KubeconfPath)
	if err != nil {
		log.Fatal(err)
	}

	kubeClient := kubernetes.NewForConfigOrDie(kubeconfig)
	k8cClient := k8cclientset.NewForConfigOrDie(kubeconfig)

	// cluster := createCluster()
	// defer deleteCluster(cluster)
	// waitForCluster(cluster)
	// installAddons(cluster)
	// clusterKubeConfig := getKubeConfig(cluster)
	// clusterClient := getClusterClient(clusterKubeConfig)
	// createMachines(clusterClient)
	// waitForMachines(clusterClient)
	// runTests(clusterKubeConfig)

	clusterTemplate := &k8cv1.Cluster{}
	clusterYaml, err := ioutil.ReadFile(runOpts.ClusterPath)
	if err != nil {
		log.Fatal(err)
	}

	clusterJSON, err := kubeyaml.ToJSON(clusterYaml)
	if err != nil {
		log.Fatal(err)
	}

	if err = json.Unmarshal(clusterJSON, clusterTemplate); err != nil {
		log.Fatal(err)
	}
	clusterTemplate.ObjectMeta.GenerateName = "e2e-test-runner-"
	clusterTemplate.ObjectMeta.Name = ""

	machineTemplate := &machinesv1alpha1.Machine{}
	machineYaml, err := ioutil.ReadFile(runOpts.MachinePath)
	if err != nil {
		log.Fatal(err)
	}

	machineJSON, err := kubeyaml.ToJSON(machineYaml)
	if err != nil {
		log.Fatal(err)
	}

	if err = json.Unmarshal(machineJSON, machineTemplate); err != nil {
		log.Fatal(err)
	}

	machineTemplate.ObjectMeta.GenerateName = clusterTemplate.ObjectMeta.GenerateName
	machineTemplate.ObjectMeta.Name = ""

	clustersCluent := k8cClient.KubermaticV1().Clusters()
	cluster, err := clustersCluent.Create(clusterTemplate)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("created cluster")

	// Cleanup after run
	defer func() {
		err1 := clustersCluent.Delete(cluster.ObjectMeta.Name, nil)
		if err1 != nil {
			log.Fatal(err)
		}
	}()

	log.Print("waiting for cluster to became healty")

	pollErr := wait.Poll(1*time.Second, runOpts.ClusterTimeout, func() (bool, error) {
		cluster, err = clustersCluent.
			Get(cluster.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		return cluster.Status.Health.ClusterHealthStatus.AllHealthy(), nil
	})
	if pollErr != nil {
		log.Fatal(pollErr)
	}

	k8cgv := k8cv1.SchemeGroupVersion
	ownerRef := *metav1.NewControllerRef(cluster, k8cgv.WithKind("Cluster"))

	for _, addon := range runOpts.Addons {
		_, err = k8cClient.KubermaticV1().
			Addons(cluster.Status.NamespaceName).
			Create(&k8cv1.Addon{
				ObjectMeta: metav1.ObjectMeta{
					Name:            addon,
					Namespace:       cluster.Status.NamespaceName,
					OwnerReferences: []metav1.OwnerReference{ownerRef},
				},
				Spec: k8cv1.AddonSpec{
					Name: addon,
					Cluster: corev1.ObjectReference{
						Name:       cluster.Name,
						Namespace:  "",
						UID:        cluster.UID,
						APIVersion: cluster.APIVersion,
						Kind:       "Cluster",
					},
				},
			})
		if err != nil {
			if kuberrrors.IsAlreadyExists(err) {
				continue
			}
			log.Fatalf("failed to create initial adddon %s for cluster %s: %v", addon, cluster.Name, err)
		}
	}

	secret, err := kubeClient.Core().
		Secrets(cluster.Status.NamespaceName).
		Get(resources.AdminKubeconfigSecretName, metav1.GetOptions{})
	if err != nil {
		log.Fatal(err)
	}
	log.Print("got admin-kubeconfig")

	clusterAdminConfig, ok := secret.Data[resources.AdminKubeconfigSecretKey]
	if !ok {
		log.Fatal("admin config not found")
	}

	clusterClientCfg, err := clientcmd.Load(clusterAdminConfig)
	restClusterConfig, err := clientcmd.NewNonInteractiveClientConfig(
		*clusterClientCfg,
		cluster.Name,
		&clientcmd.ConfigOverrides{},
		nil,
	).ClientConfig()
	if err != nil {
		log.Fatal(err)
	}

	customerClusterClient := kubernetes.NewForConfigOrDie(restClusterConfig)
	machines := machineclientset.
		NewForConfigOrDie(restClusterConfig).
		MachineV1alpha1().
		Machines()

	for i := 0; i < runOpts.Nodes; i++ {
		_, err = machines.Create(machineTemplate)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Print("waiting for machines to boot")
	pollErr = wait.Poll(1*time.Second, runOpts.NodesTimeout, func() (bool, error) {
		nodelist, err1 := customerClusterClient.
			CoreV1().
			Nodes().
			List(metav1.ListOptions{})
		if err != nil {
			return false, err1
		}

		if len(nodelist.Items) != runOpts.Nodes {
			// nodes booting
			return false, nil
		}

		for _, node := range nodelist.Items {
			for _, cnd := range node.Status.Conditions {
				if cnd.Type == corev1.NodeReady &&
					cnd.Status != corev1.ConditionTrue {
					// Kubelet didn't reported node status "Ready" yet
					return false, nil
				}
			}
		}

		return true, nil
	})

	if pollErr != nil {
		log.Fatal(pollErr)
	}

	// #########################################################################
	e2eKubeConfig, err := ioutil.TempFile("", "")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err2 := os.Remove(e2eKubeConfig.Name()); err2 != nil {
			log.Print(err2)
		}
	}()

	log.Printf("usercluster-adminconfig: %s", e2eKubeConfig.Name())

	_, err = e2eKubeConfig.Write(clusterAdminConfig)
	if err != nil {
		log.Fatal(err)
	}

	if err = e2eKubeConfig.Sync(); err != nil {
		log.Fatal(err)
	}

	execCtx, execCancel := context.WithTimeout(rootCtx, 3600*time.Second)
	cmd := exec.CommandContext(execCtx,
		runOpts.GinkgoBin,
		"--focus="+runOpts.Focus,
		"--skip="+runOpts.Skip,
		"--noColor=true",
		"--nodes="+runOpts.Parallel,
		runOpts.TestBin,
		"--",
		"--disable-log-dump",
		"--repo-root=/kubernetes",
		"--provider="+runOpts.Provider,
		"--report-dir="+runOpts.ReportsDir,
		"--kubeconfig="+e2eKubeConfig.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("running command: %v", cmd.Args)
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}

	execCancel()
}

func lookupEnv(key, defaultVal string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return defaultVal
	}
	return val
}
