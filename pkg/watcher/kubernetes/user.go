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
	userCache map[uint64]*v1.User
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
		userCache: make(map[uint64]*v1.User),
	}

	go w.run()
	return w, nil
}

func (watcher *UserWatcher) updateCache(event watch.EventType, hash uint64, user *v1.User) {
	switch event {
	case watch.Added:
	case watch.Modified:
		watcher.userCache[hash] = user
	case watch.Deleted:
		delete(watcher.userCache, hash)
	}
}

func (watcher *UserWatcher) onUserModified(hash uint64, user *v1.User) {
	cachedUser, exists := watcher.userCache[hash]
	if !exists {
		watcher.publisher.Publish(user, pubsub.LinearTreeTraverser([]uint64{hash}))
		return
	}

	// Modify lastSeen field before comparison as we want to ignore this one
	cachedUser.Status.LastSeen = user.Status.LastSeen

	if !reflect.DeepEqual(cachedUser.Spec, user.Spec) {
		watcher.publisher.Publish(user, pubsub.LinearTreeTraverser([]uint64{hash}))
	}
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

			switch event.Type {
			case watch.Added:
				watcher.publisher.Publish(user, pubsub.LinearTreeTraverser([]uint64{idHash}))
			case watch.Modified:
				watcher.onUserModified(idHash, user)
			case watch.Deleted:
				watcher.publisher.Publish(nil, pubsub.LinearTreeTraverser([]uint64{idHash}))
			}

			watcher.updateCache(event.Type, idHash, user)
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

// Subscribe allows registering subscription handler which will be invoked on each user change.
func (watcher *UserWatcher) Subscribe(subscription pubsub.Subscription, opts ...pubsub.SubscribeOption) pubsub.Unsubscriber {
	return watcher.publisher.Subscribe(subscription, opts...)
}
