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

// transportKey is used as a key for the transport cache.
// It ensures that we reuse transports only when the configuration is identical.
type transportKey struct {
	caBundle *x509.CertPool
	hostname string
}

// cacheEntry stores the transport and a reference to its element in the LRU list.
type cacheEntry struct {
	key       transportKey
	transport *http.Transport
}

var (
	// transportCache provides O(1) lookup for transports.
	transportCache = make(map[transportKey]*list.Element)
	// lruList maintains the order of usage, with the most recently used at the front.
	lruList    = list.New()
	cacheMutex = &sync.Mutex{}
)

// getTransport returns a cached or new http.Transport using an O(1) LRU cache.
// This is essential to prevent connection leaks by reusing transports.
func getTransport(hostname string, caBundle *x509.CertPool) *http.Transport {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	key := transportKey{hostname: hostname, caBundle: caBundle}

	// If a transport for this key exists, move it to the front of the list and return it.
	if element, ok := transportCache[key]; ok {
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
			lruList.Remove(lruElement)
			delete(transportCache, lruElement.Value.(*cacheEntry).key)
		}
	}

	// Add the new transport to the cache and the front of the list.
	entry := &cacheEntry{
		key:       key,
		transport: tr,
	}
	element := lruList.PushFront(entry)
	transportCache[key] = element

	return tr
}

// RemoveTransportFromCache removes a transport from the cache based on its key.
func RemoveTransportFromCache(hostname string, caBundle *x509.CertPool) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	key := transportKey{hostname: hostname, caBundle: caBundle}
	if element, ok := transportCache[key]; ok {
		lruList.Remove(element)
		delete(transportCache, key)
	}
}

// AddTransportToCache adds a given transport to the cache with a specific key.
// If the cache is full, it evicts the least recently used entry before adding the new one.
func AddTransportToCache(hostname string, caBundle *x509.CertPool, transport *http.Transport) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	key := transportKey{hostname: hostname, caBundle: caBundle}

	// If the item already exists, just move it to the front.
	if element, ok := transportCache[key]; ok {
		lruList.MoveToFront(element)
		element.Value.(*cacheEntry).transport = transport
		return
	}

	// Enforce cache size limit.
	if lruList.Len() >= maxTransportCacheSize {
		lruElement := lruList.Back()
		if lruElement != nil {
			lruList.Remove(lruElement)
			delete(transportCache, lruElement.Value.(*cacheEntry).key)
		}
	}

	// Add the new transport to the cache.
	entry := &cacheEntry{
		key:       key,
		transport: transport,
	}
	element := lruList.PushFront(entry)
	transportCache[key] = element
}

func NewClient(endpoint, accessKeyID, secretKey string, caBundle *x509.CertPool) (*minio.Client, error) {
	secure := true

	if strings.HasPrefix(endpoint, "https://") {
		endpoint = strings.Replace(endpoint, "https://", "", 1)
	} else if strings.HasPrefix(endpoint, "http://") {
		endpoint = strings.Replace(endpoint, "http://", "", 1)
		secure = false
	}

	// The hostname is used as part of the cache key for the transport.
	hostname := endpoint
	if parts := strings.Split(hostname, ":"); len(parts) > 0 {
		hostname = parts[0]
	}

	options := &minio.Options{
		Creds:     credentials.NewStaticV4(accessKeyID, secretKey, ""),
		Secure:    secure,
		Transport: getTransport(hostname, caBundle),
	}

	return minio.New(endpoint, options)
}
