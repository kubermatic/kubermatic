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
	"fmt"
	"net/http"
	"testing"
)

// resetCache is a helper function to reset the global cache state between tests.
func resetCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	transportCache = make(map[transportKey]*list.Element)
	lruList = list.New()
}

func TestGetTransport(t *testing.T) {
	caBundle1 := x509.NewCertPool()
	caBundle2 := x509.NewCertPool()

	testCases := []struct {
		name             string
		hostname         string
		caBundle         *x509.CertPool
		setup            func()
		expectedLen      int
		expectedCacheLen int
	}{
		{
			name:             "Case 1: Get a transport for the first time",
			hostname:         "s3.example.com",
			caBundle:         nil,
			setup:            resetCache,
			expectedLen:      1,
			expectedCacheLen: 1,
		},
		{
			name:     "Case 2: Get the same transport again",
			hostname: "s3.example.com",
			caBundle: nil,
			setup: func() {
				resetCache()
				getTransport("s3.example.com", nil)
			},
			expectedLen:      1,
			expectedCacheLen: 1,
		},
		{
			name:     "Case 3: Get a different transport",
			hostname: "s3.another-example.com",
			caBundle: nil,
			setup: func() {
				resetCache()
				getTransport("s3.example.com", nil)
			},
			expectedLen:      2,
			expectedCacheLen: 2,
		},
		{
			name:     "Case 4: Access the first transport again to move it to front",
			hostname: "s3.example.com",
			caBundle: nil,
			setup: func() {
				resetCache()
				getTransport("s3.example.com", nil)
				getTransport("s3.another-example.com", nil)
			},
			expectedLen:      2,
			expectedCacheLen: 2,
		},
		{
			name:             "Case 5: Get a transport with a CA bundle",
			hostname:         "s3.secure.com",
			caBundle:         caBundle1,
			setup:            resetCache,
			expectedLen:      1,
			expectedCacheLen: 1,
		},
		{
			name:     "Case 6: Get a transport with the same hostname but different CA bundle",
			hostname: "s3.secure.com",
			caBundle: caBundle2,
			setup: func() {
				resetCache()
				getTransport("s3.secure.com", caBundle1)
			},
			expectedLen:      2,
			expectedCacheLen: 2,
		},
		{
			name:     "Case 7: Get the same CA-bundled transport again",
			hostname: "s3.secure.com",
			caBundle: caBundle1,
			setup: func() {
				resetCache()
				getTransport("s3.secure.com", caBundle1)
				getTransport("s3.secure.com", caBundle2)
			},
			expectedLen:      2,
			expectedCacheLen: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			tr := getTransport(tc.hostname, tc.caBundle)
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

			if frontEntry.key.hostname != tc.hostname {
				t.Errorf("Expected hostname at front of list to be %q, but got %q", tc.hostname, frontEntry.key.hostname)
			}
		})
	}
}

func TestCacheEviction(t *testing.T) {
	t.Run("should evict the least recently used item", func(t *testing.T) {
		resetCache()

		// Fill the cache up to its maximum size.
		for i := range maxTransportCacheSize {
			hostname := fmt.Sprintf("s3.example-%d.com", i)
			getTransport(hostname, nil)
		}

		if lruList.Len() != maxTransportCacheSize {
			t.Fatalf("Expected cache to be full, size %d, but got %d", maxTransportCacheSize, lruList.Len())
		}

		// The first item added should be at the back of the list (least recently used).
		firstHost := "s3.example-0.com"
		lruKey := lruList.Back().Value.(*cacheEntry).key
		if lruKey.hostname != firstHost {
			t.Fatalf("Expected the LRU item to be %s, but got %s", firstHost, lruKey.hostname)
		}

		// Add one more transport, which should trigger an eviction.
		getTransport("s3.new-host.com", nil)

		if lruList.Len() != maxTransportCacheSize {
			t.Errorf("Expected cache size to remain %d after eviction, but got %d", maxTransportCacheSize, lruList.Len())
		}

		// Check if the least recently used item was evicted.
		keyToEvict := transportKey{hostname: firstHost, caBundle: nil}
		if _, exists := transportCache[keyToEvict]; exists {
			t.Error("The least recently used transport was not evicted from the cache")
		}
	})
}

func TestAddTransportToCache(t *testing.T) {
	t.Run("should add a custom transport and retrieve it", func(t *testing.T) {
		resetCache()

		hostname := "s3.custom.com"
		customTransport := &http.Transport{}

		// Add a custom transport.
		AddTransportToCache(hostname, nil, customTransport)

		if lruList.Len() != 1 {
			t.Fatalf("Expected list length to be 1, but got %d", lruList.Len())
		}
		if len(transportCache) != 1 {
			t.Fatalf("Expected cache size to be 1, but got %d", len(transportCache))
		}

		// Retrieve it and check if it's the same instance.
		retrieved := getTransport(hostname, nil)
		if retrieved != customTransport {
			t.Error("getTransport did not return the custom transport that was added")
		}
	})
}

func TestRemoveTransportFromCache(t *testing.T) {
	t.Run("should remove an existing transport", func(t *testing.T) {
		resetCache()

		hostname := "s3.to-remove.com"

		// Add a transport and then remove it.
		getTransport(hostname, nil)
		if lruList.Len() != 1 {
			t.Fatal("Failed to add transport to cache before removal test")
		}

		RemoveTransportFromCache(hostname, nil)

		if lruList.Len() != 0 {
			t.Errorf("Expected cache to be empty after removal, but list length is %d", lruList.Len())
		}
		if len(transportCache) != 0 {
			t.Errorf("Expected cache map to be empty after removal, but its size is %d", len(transportCache))
		}
	})

	t.Run("should not panic when removing a non-existent key", func(t *testing.T) {
		resetCache()
		// Ensure removing a non-existent key doesn't cause a panic.
		RemoveTransportFromCache("s3.non-existent.com", nil)
	})
}
