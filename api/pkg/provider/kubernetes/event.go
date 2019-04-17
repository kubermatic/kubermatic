package kubernetes

import (
	v12 "k8s.io/api/core/v1"

	"github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/scheme"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
)

const componentName = "kubermatic-api"

func NewEventRecorder() *EventRecorder {
	return &EventRecorder{seedClusterRecorderMap: make(map[string]record.EventRecorder)}
}

type EventRecorder struct {
	seedClusterRecorderMap map[string]record.EventRecorder
}

func (e *EventRecorder) SeedClusterRecorder(config *rest.Config) record.EventRecorder {
	return e.getRecorderForConfig(config)
}

func (e *EventRecorder) getRecorderForConfig(config *rest.Config) record.EventRecorder {
	recorder, exists := e.seedClusterRecorderMap[config.Host]
	if exists {
		return recorder
	}

	kubeClient := kubernetes.NewForConfigOrDie(config)
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&v1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder = eventBroadcaster.NewRecorder(scheme.Scheme, v12.EventSource{Component: componentName})
	e.seedClusterRecorderMap[config.Host] = recorder

	return recorder
}
