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
	"hash/fnv"
	"reflect"

	"code.cloudfoundry.org/go-pubsub"

	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/apimachinery/pkg/watch"
)

// UserWatcher watches user and notifies its subscribers about any changes.
type UserWatcher struct {
	provider  provider.UserProvider
	watcher   watch.Interface
	publisher *pubsub.PubSub
}

// UserWatcher returns a new resource watcher.
func NewUserWatcher(provider provider.UserProvider) (*UserWatcher, error) {
	watcher, err := provider.WatchUser()
	if err != nil {
		return nil, err
	}

	w := &UserWatcher{
		provider:  provider,
		watcher:   watcher,
		publisher: pubsub.New(),
	}

	go w.run()
	return w, nil
}

// run and publish information about user updates. Watch will restart itself if any error occurs.
func (watcher *UserWatcher) run() {
	defer func() {
		log.Logger.Debug("restarting user watcher")
		watcher.watcher.Stop()
		watcher.watcher = nil
		watcher.run()
	}()

	if watcher.watcher == nil {
		var err error
		if watcher.watcher, err = watcher.provider.WatchUser(); err != nil {
			log.Logger.Debug("could not recreate user watcher")
			return
		}
	}

	for event := range watcher.watcher.ResultChan() {
		user, ok := event.Object.(*v1.User)
		if !ok {
			log.Logger.Debugf("expected user got %s", reflect.TypeOf(event.Object))
		}

		if user != nil {
			idHash, err := watcher.CalculateHash(user.Spec.Email)
			if err != nil {
				log.Logger.Warnf("Error calculating user hash for user watch pubsub: %v", err)
				continue
			}

			if event.Type == watch.Added || event.Type == watch.Modified {
				watcher.publisher.Publish(user, pubsub.LinearTreeTraverser([]uint64{idHash}))
			} else if event.Type == watch.Deleted {
				watcher.publisher.Publish(nil, pubsub.LinearTreeTraverser([]uint64{idHash}))
			}
		}
	}
}

func (watcher *UserWatcher) CalculateHash(id string) (uint64, error) {
	h := fnv.New64()
	_, err := h.Write([]byte(id))
	if err != nil {
		return 0, err
	}
	return h.Sum64(), err
}

// Subscribe allows to register subscription handler which will be invoked on each user change.
func (watcher *UserWatcher) Subscribe(subscription pubsub.Subscription, opts ...pubsub.SubscribeOption) pubsub.Unsubscriber {
	return watcher.publisher.Subscribe(subscription, opts...)
}
