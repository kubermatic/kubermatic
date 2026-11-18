/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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
	"errors"
	"testing"

	semverlib "github.com/Masterminds/semver/v3"

	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/util/yamled"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type fakeCertManagerHelmClient struct {
	release        *helm.Release
	releaseErr     error
	maintainers    []string
	maintainersErr error
}

func (f *fakeCertManagerHelmClient) BuildChartDependencies(string, []string) error { return nil }
func (f *fakeCertManagerHelmClient) InstallChart(string, string, string, string, map[string]string, []string) error {
	return nil
}
func (f *fakeCertManagerHelmClient) GetRelease(string, string) (*helm.Release, error) {
	return f.release, f.releaseErr
}
func (f *fakeCertManagerHelmClient) GetReleaseMaintainerEmails(string, string) ([]string, error) {
	return f.maintainers, f.maintainersErr
}
func (f *fakeCertManagerHelmClient) ListReleases(string) ([]helm.Release, error) { return nil, nil }
func (f *fakeCertManagerHelmClient) UninstallRelease(string, string) error       { return nil }
func (f *fakeCertManagerHelmClient) RenderChart(string, string, string, string, map[string]string, []string) ([]byte, error) {
	return nil, nil
}
func (f *fakeCertManagerHelmClient) GetValues(string, string) (*yamled.Document, error) {
	return nil, nil
}

func TestCanManageCertManager(t *testing.T) {
	releaseWithVersion := func(version string) *helm.Release {
		return &helm.Release{Name: "cert-manager", Namespace: "cert-manager", Version: semverlib.MustParse(version)}
	}
	release := releaseWithVersion("2.31.3")
	crd := &apiextensionsv1.CustomResourceDefinition{ObjectMeta: metav1.ObjectMeta{Name: certManagerCRD}}

	testCases := []struct {
		name       string
		helmClient *fakeCertManagerHelmClient
		objects    []ctrlruntimeclient.Object
		want       bool
		wantErr    bool
	}{
		{
			name:       "nothing installed",
			helmClient: &fakeCertManagerHelmClient{},
			want:       true,
		},
		{
			name:       "release installed from the bundled chart",
			helmClient: &fakeCertManagerHelmClient{release: release, maintainers: []string{"support@kubermatic.com"}},
			objects:    []ctrlruntimeclient.Object{crd},
			want:       true,
		},
		{
			name:       "release installed from the bundled chart, oldest managed version",
			helmClient: &fakeCertManagerHelmClient{release: releaseWithVersion("2.1.0"), maintainers: []string{"support@kubermatic.com"}},
			want:       true,
		},
		{
			name:       "release installed from the bundled chart, too old (cert-manager 1.x pre-upstream chart)",
			helmClient: &fakeCertManagerHelmClient{release: releaseWithVersion("2.0.5"), maintainers: []string{"support@kubermatic.com"}},
			wantErr:    true,
		},
		{
			name:       "release installed from the bundled chart, too old (cert-manager 0.x)",
			helmClient: &fakeCertManagerHelmClient{release: releaseWithVersion("1.2.0"), maintainers: []string{"support@kubermatic.com"}},
			wantErr:    true,
		},
		{
			name:       "foreign release with an old version is skipped, not rejected",
			helmClient: &fakeCertManagerHelmClient{release: releaseWithVersion("1.17.4"), maintainers: []string{"cert-manager-maintainers@googlegroups.com"}},
			want:       false,
		},
		{
			name:       "release installed from the upstream chart",
			helmClient: &fakeCertManagerHelmClient{release: release, maintainers: []string{"cert-manager-maintainers@googlegroups.com"}},
			objects:    []ctrlruntimeclient.Object{crd},
			want:       false,
		},
		{
			name:       "release from a chart without maintainers",
			helmClient: &fakeCertManagerHelmClient{release: release, maintainers: nil},
			want:       false,
		},
		{
			name:       "CRDs without a Helm release (kubectl, operator, other release name)",
			helmClient: &fakeCertManagerHelmClient{},
			objects:    []ctrlruntimeclient.Object{crd},
			want:       false,
		},
		{
			name:       "Helm release cannot be checked",
			helmClient: &fakeCertManagerHelmClient{releaseErr: errors.New("boom")},
			wantErr:    true,
		},
		{
			name:       "release maintainers cannot be read",
			helmClient: &fakeCertManagerHelmClient{release: release, maintainersErr: errors.New("boom")},
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			if err := apiextensionsv1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			kubeClient := ctrlruntimefakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(tc.objects...).Build()

			got, reason, err := canManageCertManager(context.Background(), kubeClient, tc.helmClient)
			if (err != nil) != tc.wantErr {
				t.Fatalf("canManageCertManager() error = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("canManageCertManager() = %v, want %v", got, tc.want)
			}
			if !tc.want && !tc.wantErr && reason == "" {
				t.Fatal("expected a reason when cert-manager is not managed")
			}
		})
	}
}
