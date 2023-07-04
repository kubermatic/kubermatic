package etcd

import (
	"fmt"
	"io"

	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func closeClient(c io.Closer, log *zap.SugaredLogger) {
	err := c.Close()
	if err != nil {
		log.Warn(zap.Error(err))
	}
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func hasStrictTLS(pod *corev1.Pod) bool {
	for _, env := range pod.Spec.Containers[0].Env {
		if env.Name == "PEER_TLS_MODE" && env.Value == "strict" {
			return true
		}
	}

	return false
}

func inClusterClient() (ctrlruntimeclient.Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in cluster config: %w", err)
	}
	client, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster client: %w", err)
	}
	return client, nil
}
