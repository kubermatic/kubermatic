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
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"testing"
	"time"
)

// resetCache is a helper function to reset the global cache state between tests.
func resetCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	transportCache = make(map[string]*list.Element)
	lruList = list.New()
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
	resetCache()

	certPEM1 := newTestCertPEM(t, "org1")
	certPEM2 := newTestCertPEM(t, "org2")

	hash1 := sha256.Sum256(certPEM1)
	cacheKey1 := hex.EncodeToString(hash1[:])

	hash2 := sha256.Sum256(certPEM2)
	cacheKey2 := hex.EncodeToString(hash2[:])

	emptyHash := sha256.Sum256([]byte{})
	emptyCacheKey := hex.EncodeToString(emptyHash[:])

	testCases := []struct {
		name             string
		caBundlePEM      []byte
		setup            func()
		expectedLen      int
		expectedCacheLen int
		expectedFrontKey string
	}{
		{
			name:             "Case 1: Get a transport with nil CA bundle",
			caBundlePEM:      nil,
			setup:            resetCache,
			expectedLen:      1,
			expectedCacheLen: 1,
			expectedFrontKey: emptyCacheKey,
		},
		{
			name:        "Case 2: Get the same transport (nil CA bundle) again",
			caBundlePEM: nil,
			setup: func() {
				resetCache()
				getTransport(nil)
			},
			expectedLen:      1,
			expectedCacheLen: 1,
			expectedFrontKey: emptyCacheKey,
		},
		{
			name:        "Case 3: Get a different transport with a new CA bundle",
			caBundlePEM: certPEM1,
			setup: func() {
				resetCache()
				getTransport(nil)
			},
			expectedLen:      2,
			expectedCacheLen: 2,
			expectedFrontKey: cacheKey1,
		},
		{
			name:        "Case 4: Access an existing transport to move it to front",
			caBundlePEM: nil,
			setup: func() {
				resetCache()
				getTransport(certPEM1)
				getTransport(nil)
			},
			expectedLen:      2,
			expectedCacheLen: 2,
			expectedFrontKey: emptyCacheKey,
		},
		{
			name:             "Case 5: Get a transport with a CA bundle for the first time",
			caBundlePEM:      certPEM1,
			setup:            resetCache,
			expectedLen:      1,
			expectedCacheLen: 1,
			expectedFrontKey: cacheKey1,
		},
		{
			name:        "Case 6: Get a transport with a different CA bundle",
			caBundlePEM: certPEM2,
			setup: func() {
				resetCache()
				getTransport(certPEM1)
			},
			expectedLen:      2,
			expectedCacheLen: 2,
			expectedFrontKey: cacheKey2,
		},
		{
			name:        "Case 7: Get the same CA-bundled transport again",
			caBundlePEM: certPEM1,
			setup: func() {
				resetCache()
				getTransport(certPEM1)
				getTransport(certPEM2)
			},
			expectedLen:      2,
			expectedCacheLen: 2,
			expectedFrontKey: cacheKey1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			tr, err := getTransport(tc.caBundlePEM)
			if err != nil {
				t.Fatalf("getTransport returned an error: %v", err)
			}
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

			if frontEntry.key != tc.expectedFrontKey {
				t.Errorf("Expected front key to be %q, but got %q", tc.expectedFrontKey, frontEntry.key)
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
			getTransport(pem)
		}

		if lruList.Len() != maxTransportCacheSize {
			t.Fatalf("Expected cache to be full, size %d, but got %d", maxTransportCacheSize, lruList.Len())
		}

		// The first item added should be at the back of the list (least recently used).
		firstPEM := caBundlePEMs[0]
		hash := sha256.Sum256(firstPEM)
		expectedLruKey := hex.EncodeToString(hash[:])

		lruKey := lruList.Back().Value.(*cacheEntry).key
		if lruKey != expectedLruKey {
			t.Fatalf("Expected the LRU key to be %q, but it was %q", expectedLruKey, lruKey)
		}

		// Add one more transport, which should trigger an eviction.
		getTransport(newTestCertPEM(t, "new-org"))

		if lruList.Len() != maxTransportCacheSize {
			t.Errorf("Expected cache size to remain %d after eviction, but got %d", maxTransportCacheSize, lruList.Len())
		}

		// Check if the least recently used item was evicted.
		if _, ok := transportCache[expectedLruKey]; ok {
			t.Error("The least recently used transport was not evicted from the cache")
		}
	})
}
