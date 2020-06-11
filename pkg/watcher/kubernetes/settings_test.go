package kubernetes

import (
	"testing"

	"code.cloudfoundry.org/go-pubsub"

	kubermaticfakeclentset "github.com/kubermatic/kubermatic/pkg/crd/client/clientset/versioned/fake"
	"github.com/kubermatic/kubermatic/pkg/provider/kubernetes"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewSettingsWatcher(t *testing.T) {
	kubermaticClient := kubermaticfakeclentset.NewSimpleClientset()
	runtimeClient := fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, []runtime.Object{}...)
	settingsProvider := kubernetes.NewSettingsProvider(kubermaticClient, runtimeClient)
	settingsWatcher, err := NewSettingsWatcher(settingsProvider)
	if err != nil {
		t.Fatal("cannot create settings watcher")
	}

	counter := 0
	settingsWatcher.Subscribe(func(d interface{}) {
		counter++
	})

	if counter != 0 {
		t.Fatal("counter should be set to 0 before any data is published")
	}

	settingsWatcher.publisher.Publish("test-data", pubsub.LinearTreeTraverser([]uint64{}))

	if counter != 1 {
		t.Fatal("counter should be set to 1 after the data is published")
	}

	var data interface{}
	settingsWatcher.Subscribe(func(d interface{}) {
		data = d
	})

	settingsWatcher.publisher.Publish("test-data", pubsub.LinearTreeTraverser([]uint64{}))

	if data != "test-data" {
		t.Fatal("data should be correctly read in the subscription")
	}

	if counter != 2 {
		t.Fatal("counter should be set to 2 after the data is published for second time")
	}
}
