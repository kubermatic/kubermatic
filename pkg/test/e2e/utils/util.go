/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package utils

import (
	"context"
	"os"
	"time"

	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/defaulting"

	"k8s.io/apimachinery/pkg/util/wait"
)

func KubernetesVersion() string {
	version := defaulting.DefaultKubernetesVersioning.Default

	if v := os.Getenv("VERSION_TO_TEST"); v != "" {
		version = semver.NewSemverOrDie(v)
	}

	return "v" + version.String()
}

// WaitFor is a convenience wrapper that makes simple, "brute force"
// waiting loops easier to write.
func WaitFor(ctx context.Context, interval time.Duration, timeout time.Duration, callback func() bool) bool {
	err := wait.PollUntilContextTimeout(ctx, interval, timeout, true, func(ctx context.Context) (bool, error) {
		return callback(), nil
	})

	return err == nil
}
