/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package generator

import (
	"math/rand"
	"strings"
	"time"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ" + "0123456789")

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

type kkpSecrets struct {
	KubermaticClientSecret       string
	KubermaticIssuerClientSecret string
	IssuerCookieKey              string
	ServiceAccountKey            string
}

func generateSecrets(config Config) (kkpSecrets, error) {
	if config.GenerateSecrets {
		return kkpSecrets{
			KubermaticClientSecret:       randomString(32),
			KubermaticIssuerClientSecret: randomString(32),
			IssuerCookieKey:              randomString(32),
			ServiceAccountKey:            randomString(32),
		}, nil
	}

	return kkpSecrets{
		KubermaticClientSecret:       "<kubermatic-client-secret>",
		KubermaticIssuerClientSecret: "<kubermatic-issuer-client-secret>",
		IssuerCookieKey:              "<issuer-cookie-key>",
		ServiceAccountKey:            "<service-account-key>",
	}, nil
}

func randomString(length int) string {
	var s strings.Builder
	for i := 0; i < length; i++ {
		s.WriteRune(letterRunes[seededRand.Intn(len(letterRunes))])
	}
	return s.String()
}
