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
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"testing"
	"time"

	arccache "github.com/hashicorp/golang-lru/arc/v2"
)

// resetCache is a helper function to reset the global cache state between tests.
func resetCache() {
	var err error
	transportCache, err = arccache.NewARC[string, *http.Transport](maxTransportCacheSize)
	if err != nil {
		panic("failed to re-initialize transport cache: " + err.Error())
	}
}

// newTestCertPEM creates a new self-signed certificate and returns it as a PEM-encoded block.
func newTestCertPEM(t *testing.T, org string) []byte {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{org},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
}

func TestGetTransport(t *testing.T) {
	certPEM1 := newTestCertPEM(t, "org1")
	certPEM2 := newTestCertPEM(t, "org2")

	hash1 := sha256.Sum256(certPEM1)
	cacheKey1 := hex.EncodeToString(hash1[:])

	hash2 := sha256.Sum256(certPEM2)
	cacheKey2 := hex.EncodeToString(hash2[:])

	emptyHash := sha256.Sum256([]byte{})
	emptyCacheKey := hex.EncodeToString(emptyHash[:])

	testCases := []struct {
		name               string
		caBundlePEM        []byte
		setup              func(t *testing.T)
		expectedCacheLen   int
		expectedKeyInCache string
	}{
		{
			name:        "Case 1: Get a transport with nil CA bundle",
			caBundlePEM: nil,
			setup: func(t *testing.T) {
				resetCache()
			},
			expectedCacheLen:   1,
			expectedKeyInCache: emptyCacheKey,
		},
		{
			name:        "Case 2: Get the same transport (nil CA bundle) again",
			caBundlePEM: nil,
			setup: func(t *testing.T) {
				resetCache()
				if _, err := getTransport(nil); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			expectedCacheLen:   1,
			expectedKeyInCache: emptyCacheKey,
		},
		{
			name:        "Case 3: Get a different transport with a new CA bundle",
			caBundlePEM: certPEM1,
			setup: func(t *testing.T) {
				resetCache()
				if _, err := getTransport(nil); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			expectedCacheLen:   2,
			expectedKeyInCache: cacheKey1,
		},
		{
			name:        "Case 4: Get a transport with a CA bundle for the first time",
			caBundlePEM: certPEM1,
			setup: func(t *testing.T) {
				resetCache()
			},
			expectedCacheLen:   1,
			expectedKeyInCache: cacheKey1,
		},
		{
			name:        "Case 5: Get a transport with a different CA bundle",
			caBundlePEM: certPEM2,
			setup: func(t *testing.T) {
				resetCache()
				if _, err := getTransport(certPEM1); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			expectedCacheLen:   2,
			expectedKeyInCache: cacheKey2,
		},
		{
			name:        "Case 6: Get the same CA-bundled transport again",
			caBundlePEM: certPEM1,
			setup: func(t *testing.T) {
				resetCache()
				if _, err := getTransport(certPEM1); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				if _, err := getTransport(certPEM2); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			expectedCacheLen:   2,
			expectedKeyInCache: cacheKey1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup(t)

			tr, err := getTransport(tc.caBundlePEM)
			if err != nil {
				t.Fatalf("getTransport returned an error: %v", err)
			}
			if tr == nil {
				t.Fatal("getTransport returned a nil transport")
			}

			if transportCache.Len() != tc.expectedCacheLen {
				t.Errorf("Expected cache map size to be %d, but got %d", tc.expectedCacheLen, transportCache.Len())
			}

			// Check that the accessed element is in the cache.
			if !transportCache.Contains(tc.expectedKeyInCache) {
				t.Errorf("Expected key %q to be in the cache, but it was not", tc.expectedKeyInCache)
			}
		})
	}
}

func TestCacheEviction(t *testing.T) {
	t.Run("should evict the least recently used item", func(t *testing.T) {
		resetCache()

		// Create a slice of unique PEMs to serve as unique keys.
		caBundlePEMs := make([][]byte, maxTransportCacheSize)
		for i := range caBundlePEMs {
			caBundlePEMs[i] = newTestCertPEM(t, "org"+fmt.Sprintf("%d", i))
		}

		// Fill the cache up to its maximum size.
		for _, pem := range caBundlePEMs {
			if _, err := getTransport(pem); err != nil {
				t.Fatalf("getTransport failed during setup: %v", err)
			}
		}

		if transportCache.Len() != maxTransportCacheSize {
			t.Fatalf("Expected cache to be full, size %d, but got %d", maxTransportCacheSize, transportCache.Len())
		}

		// The first item added should be the first candidate for eviction.
		firstPEM := caBundlePEMs[0]
		hash := sha256.Sum256(firstPEM)
		keyToEvict := hex.EncodeToString(hash[:])

		// Add one more transport, which should trigger an eviction.
		if _, err := getTransport(newTestCertPEM(t, "new-org")); err != nil {
			t.Fatalf("getTransport failed during eviction test: %v", err)
		}

		if transportCache.Len() != maxTransportCacheSize {
			t.Errorf("Expected cache size to remain %d after eviction, but got %d", maxTransportCacheSize, transportCache.Len())
		}

		// Check if the least recently used item was evicted.
		if transportCache.Contains(keyToEvict) {
			t.Error("The least recently used transport was not evicted from the cache")
		}
	})
}
