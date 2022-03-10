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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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

	client, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		klog.Fatal(err)
	}

	ctx := context.Background()

	var clusterList *kubermaticv1.ClusterList
	if err := client.List(context.Background(), clusterList); err != nil {
		klog.Fatal(err)
	}

	for _, cluster := range clusterList.Items {
		if cluster.Status.NamespaceName == "" {
			klog.V(4).Infof("Skipping cluster %s because no namespaceName is set.", cluster.Name)
			continue
		}

		var ns *corev1.Namespace
		if err := client.Get(ctx, types.NamespacedName{Name: cluster.Status.NamespaceName}, ns); err != nil {
			klog.Fatal(err)
		}

		ns.OwnerReferences = []metav1.OwnerReference{}
		if err := client.Update(ctx, ns); err != nil {
			klog.Fatal(err)
		}

		var secretList *corev1.SecretList
		if err := client.List(ctx, secretList, &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}); err != nil {
			klog.Fatal(err)
		}

		for _, secret := range secretList.Items {
			secret.OwnerReferences = []metav1.OwnerReference{}
			if err := client.Update(ctx, &secret); err != nil {
				klog.Fatal(err)
			}
		}

		var configMapList *corev1.ConfigMapList
		if err := client.List(ctx, configMapList, &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}); err != nil {
			klog.Fatal(err)
		}

		for _, configMap := range configMapList.Items {
			configMap.OwnerReferences = []metav1.OwnerReference{}
			if err := client.Update(ctx, &configMap); err != nil {
				klog.Fatal(err)
			}
		}

		var serviceList *corev1.ServiceList
		if err := client.List(ctx, serviceList, &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}); err != nil {
			klog.Fatal(err)
		}

		for _, service := range serviceList.Items {
			service.OwnerReferences = []metav1.OwnerReference{}
			if err := client.Update(ctx, &service); err != nil {
				klog.Fatal(err)
			}
		}

		var pvcList *corev1.PersistentVolumeClaimList
		if err := client.List(ctx, pvcList, &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}); err != nil {
			klog.Fatal(err)
		}

		for _, pvc := range pvcList.Items {
			pvc.OwnerReferences = []metav1.OwnerReference{}
			if err := client.Update(ctx, &pvc); err != nil {
				klog.Fatal(err)
			}
		}

		var deploymentList *appsv1.DeploymentList
		if err := client.List(ctx, deploymentList, &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}); err != nil {
			klog.Fatal(err)
		}

		for _, deployment := range deploymentList.Items {
			deployment.OwnerReferences = []metav1.OwnerReference{}
			if err := client.Update(ctx, &deployment); err != nil {
				klog.Fatal(err)
			}
		}

		var statefulSetList *appsv1.StatefulSetList
		if err := client.List(ctx, statefulSetList, &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}); err != nil {
			klog.Fatal(err)
		}

		for _, statefulSet := range statefulSetList.Items {
			statefulSet.OwnerReferences = []metav1.OwnerReference{}
			if err := client.Update(ctx, &statefulSet); err != nil {
				klog.Fatal(err)
			}
		}

		var addonList *kubermaticv1.AddonList
		if err := client.List(ctx, addonList, &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}); err != nil {
			klog.Fatal(err)
		}

		for _, addon := range addonList.Items {
			addon.OwnerReferences = []metav1.OwnerReference{}
			if err := client.Update(ctx, &addon); err != nil {
				klog.Fatal(err)
			}
		}

		cmd := exec.Command("kubectl", "get", "cluster", cluster.Name, "-o", "yaml")
		out, err := cmd.CombinedOutput()
		if err != nil {
			klog.Fatal(err, string(out))
		}
		if err := os.WriteFile(fmt.Sprintf("cluster-%s.yaml", cluster.Name), out, 0644); err != nil {
			klog.Fatal(err)
		}
	}
}
