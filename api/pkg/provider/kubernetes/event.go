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

func NewEventRecorder(seedClusterConfig *rest.Config) *EventRecorder {
	recorder := &EventRecorder{
		seedClusterConfig: seedClusterConfig,
	}

	recorder.init()
	return recorder
}

type EventRecorder struct {
	seedClusterConfig *rest.Config

	seedClusterRecorder record.EventRecorder
}

func (e *EventRecorder) SeedClusterRecorder() record.EventRecorder {
	return e.seedClusterRecorder
}

func (e *EventRecorder) init() {
	kubeClient := kubernetes.NewForConfigOrDie(e.seedClusterConfig)
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&v1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	e.seedClusterRecorder = eventBroadcaster.NewRecorder(scheme.Scheme, v12.EventSource{Component: componentName})
}
