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
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
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
	key       string
	transport *http.Transport
}

var (
	// transportCache provides O(1) lookup for transports based on a string key.
	transportCache = make(map[string]*list.Element)
	// lruList maintains the order of usage, with the most recently used at the front.
	lruList    = list.New()
	cacheMutex = &sync.Mutex{}
)

// getTransport returns a cached or new http.Transport for the given CA bundle PEM.
// It uses a SHA256 hash of the PEM data as a stable cache key.
func getTransport(caBundlePEM []byte) (*http.Transport, error) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// Generate a stable key from the PEM content.
	hash := sha256.Sum256(caBundlePEM)
	cacheKey := hex.EncodeToString(hash[:])

	// If a transport for this key exists, move it to the front and return it.
	if element, ok := transportCache[cacheKey]; ok {
		lruList.MoveToFront(element)
		return element.Value.(*cacheEntry).transport, nil
	}

	// Create a new transport.
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DisableCompression = true

	// Create a CertPool and configure TLS if a bundle is provided.
	if len(caBundlePEM) > 0 {
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caBundlePEM) {
			return nil, errors.New("failed to append certificates from pem")
		}
		tr.TLSClientConfig = &tls.Config{RootCAs: caCertPool}
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

	// Add the new transport to the cache.
	entry := &cacheEntry{
		key:       cacheKey,
		transport: tr,
	}
	element := lruList.PushFront(entry)
	transportCache[cacheKey] = element

	return tr, nil
}

// NewClient creates a new S3 client using raw PEM data for the CA bundle.
func NewClient(endpoint, accessKeyID, secretKey string, caBundlePEM string) (*minio.Client, error) {
	secure := true
	if strings.HasPrefix(endpoint, "https://") {
		endpoint = strings.Replace(endpoint, "https://", "", 1)
	} else if strings.HasPrefix(endpoint, "http://") {
		endpoint = strings.Replace(endpoint, "http://", "", 1)
		secure = false
	}

	transport, err := getTransport([]byte(caBundlePEM))
	if err != nil {
		return nil, err
	}

	options := &minio.Options{
		Creds:     credentials.NewStaticV4(accessKeyID, secretKey, ""),
		Secure:    secure,
		Transport: transport,
	}

	return minio.New(endpoint, options)
}
