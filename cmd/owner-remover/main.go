package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os/exec"

	kubermaticclientset "github.com/kubermatic/kubermatic/pkg/crd/client/clientset/versioned"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

var (
	kubeconfig string
)

func main() {
	klog.InitFlags(nil)
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to the kubeconfig.")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		klog.Fatal(err)
	}

	kubermaticClient := kubermaticclientset.NewForConfigOrDie(config)
	client := kubernetes.NewForConfigOrDie(config)

	clusterList, err := kubermaticClient.KubermaticV1().Clusters().List(metav1.ListOptions{})
	if err != nil {
		klog.Fatal(err)
	}

	for _, cluster := range clusterList.Items {
		ns, err := client.CoreV1().Namespaces().Get(cluster.Status.NamespaceName, metav1.GetOptions{})
		if err != nil {
			klog.Fatal(err)
		}
		ns.OwnerReferences = []metav1.OwnerReference{}
		if _, err := client.CoreV1().Namespaces().Update(ns); err != nil {
			klog.Fatal(err)
		}

		secretList, err := client.CoreV1().Secrets(cluster.Status.NamespaceName).List(metav1.ListOptions{})
		if err != nil {
			klog.Fatal(err)
		}
		for _, secret := range secretList.Items {
			secret.OwnerReferences = []metav1.OwnerReference{}
			if _, err := client.CoreV1().Secrets(cluster.Status.NamespaceName).Update(&secret); err != nil {
				klog.Fatal(err)
			}
		}

		configMapList, err := client.CoreV1().ConfigMaps(cluster.Status.NamespaceName).List(metav1.ListOptions{})
		if err != nil {
			klog.Fatal(err)
		}
		for _, configMap := range configMapList.Items {
			configMap.OwnerReferences = []metav1.OwnerReference{}
			if _, err := client.CoreV1().ConfigMaps(cluster.Status.NamespaceName).Update(&configMap); err != nil {
				klog.Fatal(err)
			}
		}

		serviceList, err := client.CoreV1().Services(cluster.Status.NamespaceName).List(metav1.ListOptions{})
		if err != nil {
			klog.Fatal(err)
		}
		for _, service := range serviceList.Items {
			service.OwnerReferences = []metav1.OwnerReference{}
			if _, err := client.CoreV1().Services(cluster.Status.NamespaceName).Update(&service); err != nil {
				klog.Fatal(err)
			}
		}

		pvcList, err := client.CoreV1().PersistentVolumeClaims(cluster.Status.NamespaceName).List(metav1.ListOptions{})
		if err != nil {
			klog.Fatal(err)
		}
		for _, pvc := range pvcList.Items {
			pvc.OwnerReferences = []metav1.OwnerReference{}

			if _, err := client.CoreV1().PersistentVolumeClaims(cluster.Status.NamespaceName).Update(&pvc); err != nil {
				klog.Fatal(err)
			}
		}

		deploymentList, err := client.AppsV1().Deployments(cluster.Status.NamespaceName).List(metav1.ListOptions{})
		if err != nil {
			klog.Fatal(err)
		}
		for _, deployment := range deploymentList.Items {
			deployment.OwnerReferences = []metav1.OwnerReference{}
			if _, err := client.AppsV1().Deployments(cluster.Status.NamespaceName).Update(&deployment); err != nil {
				klog.Fatal(err)
			}
		}

		statefulSetList, err := client.AppsV1().StatefulSets(cluster.Status.NamespaceName).List(metav1.ListOptions{})
		if err != nil {
			klog.Fatal(err)
		}
		for _, statefulSet := range statefulSetList.Items {
			statefulSet.OwnerReferences = []metav1.OwnerReference{}
			if _, err := client.AppsV1().StatefulSets(cluster.Status.NamespaceName).Update(&statefulSet); err != nil {
				klog.Fatal(err)
			}
		}

		addonList, err := kubermaticClient.KubermaticV1().Addons(cluster.Status.NamespaceName).List(metav1.ListOptions{})
		if err != nil {
			klog.Fatal(err)
		}
		for _, addon := range addonList.Items {
			addon.OwnerReferences = []metav1.OwnerReference{}
			if _, err := kubermaticClient.KubermaticV1().Addons(cluster.Status.NamespaceName).Update(&addon); err != nil {
				klog.Fatal(err)
			}
		}

		cmd := exec.Command("kubectl", "get", "cluster", cluster.Name, "-o", "yaml")
		out, err := cmd.CombinedOutput()
		if err != nil {
			klog.Fatal(err, string(out))
		}
		if err := ioutil.WriteFile(fmt.Sprintf("cluster-%s.yaml", cluster.Name), out, 0644); err != nil {
			klog.Fatal(err)
		}
	}
}
