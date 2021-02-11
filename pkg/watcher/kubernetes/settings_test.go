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

package kubernetes

import (
	"context"
	"testing"

	"code.cloudfoundry.org/go-pubsub"

	kubermaticfakeclentset "k8c.io/kubermatic/v2/pkg/crd/client/clientset/versioned/fake"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	"k8s.io/client-go/kubernetes/scheme"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewSettingsWatcher(t *testing.T) {
	kubermaticClient := kubermaticfakeclentset.NewSimpleClientset()
	runtimeClient := fakectrlruntimeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	settingsProvider := kubernetes.NewSettingsProvider(context.Background(), kubermaticClient, runtimeClient)
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
