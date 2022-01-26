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

	"code.cloudfoundry.org/go-pubsub"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	toolscache "k8s.io/client-go/tools/cache"
)

// SettingsWatcher watches settings and notifies its subscribers about any changes.
type SettingsWatcher struct {
	log       *zap.SugaredLogger
	publisher *pubsub.PubSub
}

// SettingsWatcher returns a new resource watcher.
func NewSettingsWatcher(ctx context.Context, log *zap.SugaredLogger) (*SettingsWatcher, error) {
	w := &SettingsWatcher{
		log:       log,
		publisher: pubsub.New(),
	}

	return w, nil
}

// Subscribe allows to register subscription handler which will be invoked on each settings change.
func (watcher *SettingsWatcher) Subscribe(subscription pubsub.Subscription) pubsub.Unsubscriber {
	return watcher.publisher.Subscribe(subscription)
}

func (watcher *SettingsWatcher) OnAdd(obj interface{}) {
	watcher.onEvent(toolscache.Added, obj)
}

func (watcher *SettingsWatcher) OnUpdate(oldObj, newObj interface{}) {
	watcher.onEvent(toolscache.Updated, newObj)
}

func (watcher *SettingsWatcher) OnDelete(obj interface{}) {
	watcher.onEvent(toolscache.Deleted, obj)
}

func (watcher *SettingsWatcher) onEvent(delta toolscache.DeltaType, obj interface{}) {
	settings, ok := obj.(*kubermaticv1.KubermaticSetting)
	if !ok {
		watcher.log.Debugf("expected KubermaticSetting got %T", obj)
	}

	if settings != nil && settings.Name == kubermaticv1.GlobalSettingsName {
		if delta == toolscache.Added || delta == toolscache.Updated {
			watcher.publisher.Publish(settings, pubsub.LinearTreeTraverser([]uint64{}))
		} else if delta == toolscache.Deleted {
			watcher.publisher.Publish(nil, pubsub.LinearTreeTraverser([]uint64{}))
		}
	}
}
