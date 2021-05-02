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
	"reflect"

	"code.cloudfoundry.org/go-pubsub"

	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/apimachinery/pkg/watch"
)

// SettingsWatcher watches settings and notifies its subscribers about any changes.
type SettingsWatcher struct {
	provider  provider.SettingsProvider
	watcher   watch.Interface
	publisher *pubsub.PubSub
}

// SettingsWatcher returns a new resource watcher.
func NewSettingsWatcher(provider provider.SettingsProvider) (*SettingsWatcher, error) {
	watcher, err := provider.WatchGlobalSettings()
	if err != nil {
		return nil, err
	}

	w := &SettingsWatcher{
		provider:  provider,
		watcher:   watcher,
		publisher: pubsub.New(),
	}

	go w.run()
	return w, nil
}

// run and publish information about settings updates. Watch will restart itself if any error occur.
func (watcher *SettingsWatcher) run() {
	defer func() {
		log.Logger.Debug("restarting settings watcher")
		watcher.watcher.Stop()
		watcher.watcher = nil
		watcher.run()
	}()

	if watcher.watcher == nil {
		var err error
		if watcher.watcher, err = watcher.provider.WatchGlobalSettings(); err != nil {
			log.Logger.Debug("could not recreate settings watcher")
			return
		}
	}

	for event := range watcher.watcher.ResultChan() {
		settings, ok := event.Object.(*v1.KubermaticSetting)
		if !ok {
			log.Logger.Debugf("expected settings got %s", reflect.TypeOf(event.Object))
		}

		if settings != nil && settings.Name == v1.GlobalSettingsName {
			if event.Type == watch.Added || event.Type == watch.Modified {
				watcher.publisher.Publish(settings, pubsub.LinearTreeTraverser([]uint64{}))
			} else if event.Type == watch.Deleted {
				watcher.publisher.Publish(nil, pubsub.LinearTreeTraverser([]uint64{}))
			}
		}
	}
}

// Subscribe allows to register subscription handler which will be invoked on each settings change.
func (watcher *SettingsWatcher) Subscribe(subscription pubsub.Subscription) pubsub.Unsubscriber {
	return watcher.publisher.Subscribe(subscription)
}
