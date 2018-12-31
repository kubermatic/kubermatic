package main

import (
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	protectedNamespaces = sets.NewString(metav1.NamespaceDefault, metav1.NamespaceSystem, metav1.NamespacePublic)
)

func deleteNamespace(name string, ctx *TestContext, r *R) {
	err := wait.Poll(10*time.Second, 10*time.Minute, func() (done bool, err error) {
		ns, err := ctx.clusterContext.kubeClient.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, nil
		}

		if ns.DeletionTimestamp != nil {
			return false, nil
		}

		ctx.clusterContext.kubeClient.CoreV1().Namespaces().Delete(name, nil)
		return false, nil
	})

	if err != nil {
		r.Errorf("failed to cleanup all namespaces: %v", err)
		return
	}
}

func deleteAllNonDefaultNamespaces(ctx *TestContext, r *R) {
	err := wait.Poll(10*time.Second, 10*time.Minute, func() (done bool, err error) {
		namespaceList, err := ctx.clusterContext.kubeClient.CoreV1().Namespaces().List(metav1.ListOptions{})
		if err != nil {
			r.Logf("failed to list namespaces: %v", err)
			return false, nil
		}

		// This check assumes no one deleted one of the protected namespaces
		if len(namespaceList.Items) <= protectedNamespaces.Len() {
			return true, nil
		}

		for _, namespace := range namespaceList.Items {
			if protectedNamespaces.Has(namespace.Name) {
				continue
			}
			// If its not gone & the DeletionTimestamp is nil, delete it
			if namespace.DeletionTimestamp == nil {
				if err := ctx.clusterContext.kubeClient.CoreV1().Namespaces().Delete(namespace.Name, nil); err != nil {
					r.Logf("Failed to delete namespace: %v", err)
				}
			}
		}
		return false, nil
	})

	if err != nil {
		r.Errorf("failed to cleanup all namespaces: %v", err)
		return
	}
}
