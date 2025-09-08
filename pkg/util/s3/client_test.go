/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package s3

import (
	"container/list"
	"crypto/x509"
	"net/http"
	"testing"
)

// resetCache is a helper function to reset the global cache state between tests.
func resetCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	transportCache = make(map[*x509.CertPool]*list.Element)
	lruList = list.New()
}

func TestGetTransport(t *testing.T) {
	caBundle1 := x509.NewCertPool()
	caBundle2 := x509.NewCertPool()

	testCases := []struct {
		name             string
		caBundle         *x509.CertPool
		setup            func()
		expectedLen      int
		expectedCacheLen int
	}{
		{
			name:             "Case 1: Get a transport with nil CA bundle",
			caBundle:         nil,
			setup:            resetCache,
			expectedLen:      1,
			expectedCacheLen: 1,
		},
		{
			name:     "Case 2: Get the same transport (nil CA bundle) again",
			caBundle: nil,
			setup: func() {
				resetCache()
				getTransport(nil)
			},
			expectedLen:      1,
			expectedCacheLen: 1,
		},
		{
			name:     "Case 3: Get a different transport with a new CA bundle",
			caBundle: caBundle1,
			setup: func() {
				resetCache()
				getTransport(nil)
			},
			expectedLen:      2,
			expectedCacheLen: 2,
		},
		{
			name:     "Case 4: Access an existing transport to move it to front",
			caBundle: nil,
			setup: func() {
				resetCache()
				getTransport(nil)
				getTransport(caBundle1)
			},
			expectedLen:      2,
			expectedCacheLen: 2,
		},
		{
			name:             "Case 5: Get a transport with a CA bundle for the first time",
			caBundle:         caBundle1,
			setup:            resetCache,
			expectedLen:      1,
			expectedCacheLen: 1,
		},
		{
			name:     "Case 6: Get a transport with a different CA bundle",
			caBundle: caBundle2,
			setup: func() {
				resetCache()
				getTransport(caBundle1)
			},
			expectedLen:      2,
			expectedCacheLen: 2,
		},
		{
			name:     "Case 7: Get the same CA-bundled transport again",
			caBundle: caBundle1,
			setup: func() {
				resetCache()
				getTransport(caBundle1)
				getTransport(caBundle2)
			},
			expectedLen:      2,
			expectedCacheLen: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			tr := getTransport(tc.caBundle)
			if tr == nil {
				t.Fatal("getTransport returned a nil transport")
			}

			if lruList.Len() != tc.expectedLen {
				t.Errorf("Expected LRU list length to be %d, but got %d", tc.expectedLen, lruList.Len())
			}
			if len(transportCache) != tc.expectedCacheLen {
				t.Errorf("Expected cache map size to be %d, but got %d", tc.expectedCacheLen, len(transportCache))
			}

			// Check that the accessed element is now at the front of the list.
			frontElement := lruList.Front()
			if frontElement == nil {
				if tc.expectedLen > 0 {
					t.Fatal("LRU list is unexpectedly empty")
				}
				return // List is empty as expected.
			}

			frontEntry, ok := frontElement.Value.(*cacheEntry)
			if !ok {
				t.Fatal("Front element of LRU list has an invalid type")
			}

			expectedKey := tc.caBundle
			if frontEntry.key != expectedKey {
				t.Errorf("Expected caBundle pointer at front of list to be %p, but got %p", expectedKey, frontEntry.key)
			}
		})
	}
}

func TestCacheEviction(t *testing.T) {
	t.Run("should evict the least recently used item", func(t *testing.T) {
		resetCache()

		// Create a slice of unique CertPools to serve as unique keys.
		caBundles := make([]*x509.CertPool, maxTransportCacheSize)
		for i := range caBundles {
			caBundles[i] = x509.NewCertPool()
		}

		// Fill the cache up to its maximum size.
		for _, bundle := range caBundles {
			getTransport(bundle)
		}

		if lruList.Len() != maxTransportCacheSize {
			t.Fatalf("Expected cache to be full, size %d, but got %d", maxTransportCacheSize, lruList.Len())
		}

		// The first item added should be at the back of the list (least recently used).
		firstBundle := caBundles[0]
		lruKey := lruList.Back().Value.(*cacheEntry).key
		if lruKey != firstBundle {
			t.Fatalf("Expected the LRU item to have pointer %p, but got %p", firstBundle, lruKey)
		}

		// Add one more transport, which should trigger an eviction.
		getTransport(x509.NewCertPool())

		if lruList.Len() != maxTransportCacheSize {
			t.Errorf("Expected cache size to remain %d after eviction, but got %d", maxTransportCacheSize, lruList.Len())
		}

		// Check if the least recently used item was evicted.
		keyToEvict := firstBundle
		if _, exists := transportCache[keyToEvict]; exists {
			t.Error("The least recently used transport was not evicted from the cache")
		}
	})
}

func TestAddTransportToCache(t *testing.T) {
	t.Run("should add a custom transport and retrieve it", func(t *testing.T) {
		resetCache()

		customTransport := &http.Transport{}

		// Add a custom transport.
		AddTransportToCache(nil, customTransport)

		if lruList.Len() != 1 {
			t.Fatalf("Expected list length to be 1, but got %d", lruList.Len())
		}
		if len(transportCache) != 1 {
			t.Fatalf("Expected cache size to be 1, but got %d", len(transportCache))
		}

		// Retrieve it and check if it's the same instance.
		retrieved := getTransport(nil)
		if retrieved != customTransport {
			t.Error("getTransport did not return the custom transport that was added")
		}
	})

	t.Run("should move an existing transport to the front when updated", func(t *testing.T) {
		resetCache()
		caBundle := x509.NewCertPool()

		// Add two transports.
		AddTransportToCache(nil, &http.Transport{})
		AddTransportToCache(caBundle, &http.Transport{}) // This is now the front.

		// Check that the second transport is at the front.
		frontKey := lruList.Front().Value.(*cacheEntry).key
		if frontKey != caBundle {
			t.Fatal("Test setup failed: The second transport was not at the front of the list.")
		}

		// Update the first transport, which should move it to the front.
		updatedTransport := &http.Transport{}
		AddTransportToCache(nil, updatedTransport)

		if lruList.Len() != 2 {
			t.Fatalf("Expected list length to remain 2, but got %d", lruList.Len())
		}

		// Verify that the updated transport is now at the front.
		newFrontEntry := lruList.Front().Value.(*cacheEntry)
		if newFrontEntry.key != nil {
			t.Error("Expected the updated transport's key to be at the front of the list")
		}
		if newFrontEntry.transport != updatedTransport {
			t.Error("The transport in the cache was not updated to the new instance")
		}
	})

	t.Run("should evict the least recently used item when cache is full", func(t *testing.T) {
		resetCache()

		// Fill the cache to capacity.
		caBundles := make([]*x509.CertPool, maxTransportCacheSize)
		for i := range caBundles {
			caBundles[i] = x509.NewCertPool()
			AddTransportToCache(caBundles[i], &http.Transport{})
		}

		if lruList.Len() != maxTransportCacheSize {
			t.Fatalf("Expected cache to be full, size %d, but got %d", maxTransportCacheSize, lruList.Len())
		}

		// The first item added should be at the back (LRU).
		firstBundle := caBundles[0]
		lruKey := lruList.Back().Value.(*cacheEntry).key
		if lruKey != firstBundle {
			t.Fatalf("Expected the LRU item to have the pointer for the first bundle, but it did not.")
		}

		// Add one more item to trigger eviction.
		AddTransportToCache(x509.NewCertPool(), &http.Transport{})

		if lruList.Len() != maxTransportCacheSize {
			t.Errorf("Expected cache size to remain %d after eviction, but got %d", maxTransportCacheSize, lruList.Len())
		}

		// Check that the LRU item was evicted.
		keyToEvict := firstBundle
		if _, exists := transportCache[keyToEvict]; exists {
			t.Error("The least recently used transport was not evicted from the cache")
		}
	})
}

func TestRemoveTransportFromCache(t *testing.T) {
	t.Run("should remove an existing transport", func(t *testing.T) {
		resetCache()
		caBundle := x509.NewCertPool()

		// Add two transports.
		getTransport(nil)
		getTransport(caBundle)

		if lruList.Len() != 2 {
			t.Fatalf("Expected cache to have 2 items before removal, but got %d", lruList.Len())
		}

		// Remove the first transport (which is now the least recently used).
		RemoveTransportFromCache(nil)

		if lruList.Len() != 1 {
			t.Errorf("Expected cache to have 1 item after removal, but list length is %d", lruList.Len())
		}
		if len(transportCache) != 1 {
			t.Errorf("Expected cache map to have 1 item after removal, but its size is %d", len(transportCache))
		}

		// Check that the correct transport was removed and the other one remains.
		keyToRemove := (*x509.CertPool)(nil)
		if _, exists := transportCache[keyToRemove]; exists {
			t.Error("The specified transport was not removed from the cache map")
		}

		remainingKey := lruList.Front().Value.(*cacheEntry).key
		if remainingKey != caBundle {
			t.Error("The wrong transport was removed from the cache")
		}
	})

	t.Run("should not panic when removing a non-existent key", func(t *testing.T) {
		resetCache()
		// Ensure removing a non-existent key doesn't cause a panic.
		RemoveTransportFromCache(nil)
	})
}
