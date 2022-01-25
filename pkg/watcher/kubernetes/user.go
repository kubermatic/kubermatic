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
	"hash/fnv"
	"reflect"

	"code.cloudfoundry.org/go-pubsub"
	"go.uber.org/zap"

	v1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/watch"
	toolscache "k8s.io/client-go/tools/cache"
)

// UserWatcher watches user and notifies its subscribers about any changes.
type UserWatcher struct {
	log       *zap.SugaredLogger
	publisher *pubsub.PubSub
	userCache map[uint64]*v1.User
}

var _ toolscache.ResourceEventHandler = &UserWatcher{}

// UserWatcher returns a new resource watcher.
func NewUserWatcher(ctx context.Context, log *zap.SugaredLogger) (*UserWatcher, error) {
	w := &UserWatcher{
		log:       log,
		publisher: pubsub.New(),
		userCache: make(map[uint64]*v1.User),
	}

	return w, nil
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

func (watcher *UserWatcher) OnAdd(obj interface{}) {
	user, ok := obj.(*v1.User)
	if !ok {
		watcher.log.Debugf("expected User but got %T", obj)
		return
	}

	idHash, err := watcher.CalculateHash(user.Spec.Email)
	if err != nil {
		watcher.log.Warnf("Error calculating user hash for user watch pubsub: %v", err)
		return
	}

	watcher.publisher.Publish(user, pubsub.LinearTreeTraverser([]uint64{idHash}))
	watcher.updateCache(watch.Added, idHash, user)
}

func (watcher *UserWatcher) OnUpdate(oldObj, newObj interface{}) {
	user, ok := newObj.(*v1.User)
	if !ok {
		watcher.log.Debugf("expected User but got %T", newObj)
		return
	}

	idHash, err := watcher.CalculateHash(user.Spec.Email)
	if err != nil {
		watcher.log.Warnf("Error calculating user hash for user watch pubsub: %v", err)
		return
	}

	cachedUser, exists := watcher.userCache[idHash]
	if !exists {
		watcher.publisher.Publish(user, pubsub.LinearTreeTraverser([]uint64{idHash}))
		return
	}

	// Modify lastSeen field before comparison as we want to ignore this one
	cachedUser.Status.LastSeen = user.Status.LastSeen

	if !reflect.DeepEqual(cachedUser.Spec, user.Spec) {
		watcher.publisher.Publish(user, pubsub.LinearTreeTraverser([]uint64{idHash}))
	}

	watcher.updateCache(watch.Modified, idHash, user)
}

func (watcher *UserWatcher) OnDelete(obj interface{}) {
	user, ok := obj.(*v1.User)
	if !ok {
		watcher.log.Debugf("expected User but got %T", obj)
		return
	}

	idHash, err := watcher.CalculateHash(user.Spec.Email)
	if err != nil {
		watcher.log.Warnf("Error calculating user hash for user watch pubsub: %v", err)
		return
	}

	watcher.publisher.Publish(nil, pubsub.LinearTreeTraverser([]uint64{idHash}))
	watcher.updateCache(watch.Deleted, idHash, user)
}

func (watcher *UserWatcher) updateCache(event watch.EventType, hash uint64, user *v1.User) {
	switch event {
	case watch.Added:
		fallthrough
	case watch.Modified:
		watcher.userCache[hash] = user
	case watch.Deleted:
		delete(watcher.userCache, hash)
	}
}
