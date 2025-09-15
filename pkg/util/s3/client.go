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
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	arccache "github.com/hashicorp/golang-lru/arc/v2"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	// maxTransportCacheSize defines the maximum number of idle transports to keep in the cache.
	maxTransportCacheSize = 30
)

var (
	// transportCache provides a thread-safe, fixed-size ARC cache for http.Transport objects.
	transportCache *arccache.ARCCache[string, *http.Transport]
)

func init() {
	var err error
	transportCache, err = arccache.NewARC[string, *http.Transport](maxTransportCacheSize)
	if err != nil {
		// This should not happen with a static size > 0.
		panic("failed to initialize transport cache: " + err.Error())
	}
}

// getTransport returns a cached or new http.Transport for the given CA bundle PEM.
// It uses a SHA256 hash of the PEM data as a stable cache key.
func getTransport(caBundlePEM []byte) (*http.Transport, error) {
	// Generate a stable key from the PEM content.
	hash := sha256.Sum256(caBundlePEM)
	cacheKey := hex.EncodeToString(hash[:])

	// If a transport for this key exists, return it.
	if tr, ok := transportCache.Get(cacheKey); ok {
		return tr, nil
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

	// Add the new transport to the cache.
	transportCache.Add(cacheKey, tr)

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
