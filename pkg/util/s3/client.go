/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"strings"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	// maxTransportCacheSize defines the maximum number of idle transports to keep in the cache.
	maxTransportCacheSize = 30
)

// cacheEntry stores the transport and a reference to its element in the LRU list.
type cacheEntry struct {
	key       *x509.CertPool
	transport *http.Transport
}

var (
	// transportCache provides O(1) lookup for transports.
	// Using a pointer (*x509.CertPool) as a key is acceptable here because the cache's purpose
	// is to tie a transport to a specific CertPool instance. The memory address of the CertPool
	// serves as a unique identifier. This assumes that different CA bundles that require
	// distinct transports will be represented by separate CertPool objects in memory.
	// A nil pointer is a valid map key and all nil keys will map to the same entry, which is
	// the desired behavior for sharing a default transport.
	transportCache = make(map[*x509.CertPool]*list.Element)
	// lruList maintains the order of usage, with the most recently used at the front.
	lruList    = list.New()
	cacheMutex = &sync.Mutex{}
)

// getTransport returns a cached or new http.Transport using an O(1) LRU cache.
// This is essential to prevent connection leaks by reusing transports.
func getTransport(caBundle *x509.CertPool) *http.Transport {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// If a transport for this caBundle exists, move it to the front of the list and return it.
	if element, ok := transportCache[caBundle]; ok {
		lruList.MoveToFront(element)
		return element.Value.(*cacheEntry).transport
	}

	// Create a new transport, cloning the default to inherit basic settings.
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DisableCompression = true

	// If a caBundle is provided, configure TLS. Otherwise, it will use system defaults.
	if caBundle != nil {
		tr.TLSClientConfig = &tls.Config{RootCAs: caBundle}
	}

	// Enforce cache size limit by evicting the least recently used item.
	if lruList.Len() >= maxTransportCacheSize {
		lruElement := lruList.Back()
		if lruElement != nil {
			evictedKey := lruElement.Value.(*cacheEntry).key
			lruList.Remove(lruElement)
			delete(transportCache, evictedKey)
		}
	}

	// Add the new transport to the cache and the front of the list.
	entry := &cacheEntry{
		key:       caBundle,
		transport: tr,
	}
	element := lruList.PushFront(entry)
	transportCache[caBundle] = element

	return tr
}

// RemoveTransportFromCache removes a transport from the cache based on its key.
func RemoveTransportFromCache(caBundle *x509.CertPool) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	if element, ok := transportCache[caBundle]; ok {
		lruList.Remove(element)
		delete(transportCache, caBundle)
	}
}

// AddTransportToCache adds a given transport to the cache with a specific key.
// If the cache is full, it evicts the least recently used entry before adding the new one.
func AddTransportToCache(caBundle *x509.CertPool, transport *http.Transport) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// If the item already exists, just move it to the front.
	if element, ok := transportCache[caBundle]; ok {
		lruList.MoveToFront(element)
		element.Value.(*cacheEntry).transport = transport
		return
	}

	// Enforce cache size limit.
	if lruList.Len() >= maxTransportCacheSize {
		lruElement := lruList.Back()
		if lruElement != nil {
			evictedKey := lruElement.Value.(*cacheEntry).key
			lruList.Remove(lruElement)
			delete(transportCache, evictedKey)
		}
	}

	// Add the new transport to the cache.
	entry := &cacheEntry{
		key:       caBundle,
		transport: transport,
	}
	element := lruList.PushFront(entry)
	transportCache[caBundle] = element
}

func NewClient(endpoint, accessKeyID, secretKey string, caBundle *x509.CertPool) (*minio.Client, error) {
	secure := true

	if strings.HasPrefix(endpoint, "https://") {
		endpoint = strings.Replace(endpoint, "https://", "", 1)
	} else if strings.HasPrefix(endpoint, "http://") {
		endpoint = strings.Replace(endpoint, "http://", "", 1)
		secure = false
	}
	// secure = false
	options := &minio.Options{
		Creds:     credentials.NewStaticV4(accessKeyID, secretKey, ""),
		Secure:    secure,
		Transport: getTransport(caBundle),
	}

	return minio.New(endpoint, options)
}
