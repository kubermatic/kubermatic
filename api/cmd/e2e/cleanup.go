package main

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	protectedNamespaces = sets.NewString(metav1.NamespaceDefault, metav1.NamespaceSystem, metav1.NamespacePublic)
)

func deleteNamespace(name string, ctx *TestContext, r *R) {
	err := wait.Poll(10*time.Second, 10*time.Minute, func() (done bool, err error) {
		ns := &corev1.Namespace{}
		if err := ctx.clusterContext.client.Get(context.Background(), types.NamespacedName{Name: name}, ns); err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, nil
		}
		if ns.DeletionTimestamp != nil {
			return false, nil
		}

		ctx.clusterContext.client.Delete(context.Background(), ns, nil)
		return false, nil
	})

	if err != nil {
		r.Errorf("failed to cleanup all namespaces: %v", err)
		return
	}
}
