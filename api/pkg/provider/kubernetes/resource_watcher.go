package kubernetes

import (
	"fmt"
	"reflect"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"code.cloudfoundry.org/go-pubsub"
	"k8s.io/apimachinery/pkg/watch"
)

// ResourceWatcher watches resources and notifies its subscribers about any changes.
type ResourceWatcher struct {
	settingsProvider provider.SettingsProvider
	publisher        *pubsub.PubSub
	settingsWatcher  watch.Interface
}

// ResourceWatcher returns a new resource watcher.
func NewResourceWatcher(settingsProvider provider.SettingsProvider) (*ResourceWatcher, error) {
	settingsWatcher, err := settingsProvider.WatchGlobalSettings()
	if err != nil {
		return nil, err
	}

	publisher := pubsub.New()
	go watchGlobalSettings(settingsWatcher, publisher)

	subscription := func(name string) func(interface{}) {
		return func(data interface{}) {
			fmt.Printf("%s -> %v\n", name, data)
		}
	}

	publisher.Subscribe(subscription("debugSub"))
	fmt.Println("test123")

	return &ResourceWatcher{
		settingsProvider: settingsProvider,
		settingsWatcher:  settingsWatcher,
		publisher:        publisher,
	}, nil
}

func watchGlobalSettings(input watch.Interface, publisher *pubsub.PubSub) {
	defer input.Stop()

	for {
		select {
		case event, ok := <-input.ResultChan():
			if !ok {
				fmt.Println("watch ended with timeout")
				return
			}

			var settings *kubermaticv1.KubermaticSetting

			switch event.Type {
			case watch.Added:
				settings, ok = event.Object.(*kubermaticv1.KubermaticSetting)
				if !ok {
					fmt.Printf("expected settings got %s", reflect.TypeOf(event.Object))
				}
			case watch.Modified:
				settings, ok = event.Object.(*kubermaticv1.KubermaticSetting)
				if !ok {
					fmt.Printf("expected settings got %s", reflect.TypeOf(event.Object))
				}
			case watch.Deleted:
				settings = nil
			case watch.Error:
				fmt.Println("error")
			}

			fmt.Println("asd")
			fmt.Println(settings.Name)
			fmt.Println(settings.Namespace)
			fmt.Println(settings)

			if settings != nil && settings.Name == kubermaticv1.GlobalSettingsName { // TODO namespace
				publisher.Publish(settings, pubsub.LinearTreeTraverser([]uint64{1}))
			}
		}
	}
}

func (watcher *ResourceWatcher) WatchKubermaticSettings(subscription pubsub.Subscription) {
	watcher.publisher.Subscribe(subscription)
}
