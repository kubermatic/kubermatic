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

package kubermaticseed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/install/stack/common"
	"k8c.io/kubermatic/v2/pkg/util/podexec"
	"k8c.io/kubermatic/v2/pkg/util/yamled"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (*SeedStack) ValidateState(ctx context.Context, opt stack.DeployOptions) []error {
	failures := []error{}

	if err := ValidateMinioCompatibility(ctx, opt); err != nil {
		failures = append(failures, err)
	}

	return failures
}

// lastCompatibleMinioRelease is the most recent Minio release that still contains
// support for the legacy fs driver.
// See https://github.com/minio/minio/releases/tag/RELEASE.2022-10-29T06-21-33Z
// changelog and Minio PR 15929 for the removal info in the next release.
var lastCompatibleMinioRelease = "RELEASE.2022-10-24T18-35-07Z"

// In KKP 2.23, Minio RELEASE.2023-05-04T21-44-30Z i shipped. This version breaks compat
// with previous versions as the legacy "fs" filesystem driver has been removed.
// Since Minio RELEASE.2022-06-25T15-50-16Z (KKP 2.21), the default filesystem driver
// was "xl" already, the new replacement for "fs".
// This means any Minio [PVC] that was created with KKP 2.21+ is forward-compatible,
// any PVC originally created with older Minio releases however will not survive the
// KKP 2.23 upgrade, as a manual migration is required.
// See https://github.com/kubermatic/kubermatic/issues/12430 for more information.
// This function will validate Minio's currently used filesystem driver and report
// an error if upgrading won't be possible.
func ValidateMinioCompatibility(ctx context.Context, opt stack.DeployOptions) error {
	// The last Minio release that can still handle "fs" storage is RELEASE.2022-10-24T18-35-07Z;
	// if the user has configured this or any older version explicitly in their Helm values,
	// we do not need to perform any further checks and can save a lot of work.
	minioTag, ok := opt.HelmValues.GetString(yamled.Path{"minio", "image", "tag"})
	if ok {
		if !strings.HasPrefix(minioTag, "RELEASE.") {
			opt.Logger.WithField("tag", minioTag).Warn("Cannot parse customized Minio tag, cannot skip PVC compatibility check")
		} else if minioTag <= lastCompatibleMinioRelease {
			return nil // an old release is configured, nothing can go wrong
		}
	}

	release, err := opt.HelmClient.GetRelease(MinioNamespace, MinioReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check Helm releases: %w", err)
	}

	// Minio has not been installed (yet?); perfect, the user is free to
	// install whatever version they wish.
	if release == nil {
		return nil
	}

	// Checking compatibility requires to actually inspect Minio's filesystem;
	// the current Helm release version won't tell us the original version that
	// created the storage, and Minio's Admin API does not provide the filesystem
	// driver name.
	pods := corev1.PodList{}
	if err := opt.KubeClient.List(ctx, &pods, &ctrlruntimeclient.ListOptions{
		Namespace: MinioNamespace,
		LabelSelector: labels.ValidatedSetSelector{
			"app": "minio",
		},
	}); err != nil {
		return fmt.Errorf("failed to find Minio pod: %w", err)
	}

	// As the Helm chart provisions a singular PVC, we expect a singular pod. There
	// is no option in the chart to configure a GCS/S3-backed Minio proxy that might
	// run multiple replicas.
	if len(pods.Items) != 1 {
		return fmt.Errorf("expected exactly 1 Minio Pod, but found %d; cannot exec and check PVC contents", len(pods.Items))
	}

	minioPod := ctrlruntimeclient.ObjectKeyFromObject(&pods.Items[0])

	// Exec into the pod and look under Minio's hood.
	command := []string{"cat", "/storage/.minio.sys/format.json"}
	stdout, _, err := podexec.ExecuteCommand(ctx, opt.RestConfig, minioPod, "minio", command...)
	if err != nil {
		return fmt.Errorf("failed to execute command in Minio container: %w", err)
	}

	// parse Minio's config file
	type minioFormat struct {
		Format string `json:"format"`
	}

	data := minioFormat{}
	if err := json.Unmarshal([]byte(stdout), &data); err != nil {
		return fmt.Errorf("failed to decode %q as JSON: %w", stdout, err)
	}

	// Bad news: This Minio is using the old, legacy fs driver and needs to be migrated manually.
	if data.Format == "fs" {
		return errors.New("the Minio storage is using the `fs` filesystem driver, which is incompatible with more recent Minio releases and requires a migration; please refer to https://docs.kubermatic.com/kubermatic/v2.23/installation/upgrading/upgrade-from-2.22-to-2.23/#minio-upgrade for more information")
	}

	// Good news, the storage is probably using "xl" and so it's future-ready.
	return nil
}

func (*SeedStack) ValidateConfiguration(config *kubermaticv1.KubermaticConfiguration, helmValues *yamled.Document, opt stack.DeployOptions, logger logrus.FieldLogger) (*kubermaticv1.KubermaticConfiguration, *yamled.Document, []error) {
	helmFailures := validateHelmValues(helmValues)
	for idx, e := range helmFailures {
		helmFailures[idx] = prefixError("Helm values: ", e)
	}

	return config, helmValues, helmFailures
}

func validateHelmValues(helmValues *yamled.Document) []error {
	if helmValues.IsEmpty() {
		return []error{fmt.Errorf("No Helm Values file was provided, or the file was empty; installation cannot proceed. Please use the flag --helm-values=<valuesfile.yaml>")}
	}

	failures := []error{}

	path := yamled.Path{"minio", "credentials", "accessKey"}
	accessKey, _ := helmValues.GetString(path)
	if err := common.ValidateRandomSecret(accessKey, path.String()); err != nil {
		failures = append(failures, err)
	}

	path = yamled.Path{"minio", "credentials", "secretKey"}
	secretKey, _ := helmValues.GetString(path)
	if err := common.ValidateRandomSecret(secretKey, path.String()); err != nil {
		failures = append(failures, err)
	}

	return failures
}

func prefixError(prefix string, e error) error {
	return fmt.Errorf("%s%w", prefix, e)
}
