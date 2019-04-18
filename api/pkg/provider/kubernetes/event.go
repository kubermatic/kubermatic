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

// NewEventRecorder returns a new event recorder provider object. See EventRecorder for more information.
func NewEventRecorder() *EventRecorder {
	return &EventRecorder{seedClusterRecorderMap: make(map[string]record.EventRecorder)}
}

// EventRecorder gives option to record events for objects. They can be then read from them using K8S API.
type EventRecorder struct {
	seedClusterRecorderMap map[string]record.EventRecorder
}

// SeedClusterRecorder returns a event recorder that will be able to record event for objects in the cluster
// referred by provided cluster config.
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
