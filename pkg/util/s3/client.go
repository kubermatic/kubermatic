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
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"strings"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// transportKey is used as a key for the transport cache.
// It ensures that we reuse transports only when the configuration is identical.
type transportKey struct {
	caBundle *x509.CertPool
	hostname string
}

var (
	transportCache = make(map[transportKey]*http.Transport)
	cacheMutex     = &sync.Mutex{}
)

// getTransport returns a cached or new http.Transport.
// This is essential to prevent connection leaks by reusing transports.
func getTransport(hostname string, caBundle *x509.CertPool) *http.Transport {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	key := transportKey{hostname: hostname, caBundle: caBundle}

	// If a transport for this exact key already exists, reuse it.
	if transport, ok := transportCache[key]; ok {
		return transport
	}

	// Create a new transport, cloning the default to inherit basic settings.
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DisableCompression = true

	// If a caBundle is provided, configure TLS. Otherwise, it will use system defaults.
	if caBundle != nil {
		tr.TLSClientConfig = &tls.Config{RootCAs: caBundle}
	}

	// Cache the newly created transport.
	transportCache[key] = tr

	return tr
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
