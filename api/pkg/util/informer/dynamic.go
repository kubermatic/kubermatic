package informer

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"

	ctrlruntimecache "sigs.k8s.io/controller-runtime/pkg/cache"
)

const (
	informerSyncTimeout = 5 * time.Second
)

// GetSyncedStoreFromDynamicFactory returns a synced informer for the given type.
// The informer must sync within 5 seconds, otherwise an error will be returned
func GetSyncedStoreFromDynamicFactory(factory ctrlruntimecache.Cache, obj runtime.Object) (cache.Store, error) {
	informer, err := factory.GetInformer(obj)
	if err != nil {
		return nil, err
	}

	store := informer.GetStore()
	// If possible, we avoid the WaitForCacheSync block as it creates some noise in the logs
	if informer.GetController().HasSynced() {
		return store, nil
	}

	if !cache.WaitForCacheSync(getDefaultInformerSyncStopCh(), informer.GetController().HasSynced) {
		return nil, fmt.Errorf("timed out while waiting for the Informer to sync")
	}

	return store, nil
}

// getDefaultInformerSyncStopCh returns a stop channel which will be closed after 5 seconds.
// Useful for informer synchronization
func getDefaultInformerSyncStopCh() <-chan struct{} {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(informerSyncTimeout)
		cancel()
	}()
	return ctx.Done()
}
