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

package manifests

import (
	"embed"
	"path"
	"reflect"
	"testing"

	kubevirtv1 "kubevirt.io/api/core/v1"
	kvinstancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	fakeclient ctrlruntimeclient.Client

	//go:embed test/instancetypes
	tesInstancetypeFS   embed.FS
	testInstancetypeDir = path.Join("test", "instancetypes")

	//go:embed test/preferences
	tesPreferenceFS   embed.FS
	testPreferenceDir = path.Join("test", "preferences")
)

func init() {
	utilruntime.Must(kubevirtv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(kvinstancetypev1alpha1.AddToScheme(scheme.Scheme))
	fakeclient = ctrlruntimefakeclient.
		NewClientBuilder().
		WithScheme(scheme.Scheme).
		Build()
}

type testInstancetypeGetter ManifestFS

func (s *testInstancetypeGetter) GetManifestFS() *ManifestFS {
	return &ManifestFS{
		Fs:  tesInstancetypeFS,
		Dir: testInstancetypeDir,
	}
}

type testPreferenceGetter ManifestFS

func (s *testPreferenceGetter) GetManifestFS() *ManifestFS {
	return &ManifestFS{
		Fs:  tesPreferenceFS,
		Dir: testPreferenceDir,
	}
}

func TestKubernetesFromYaml(t *testing.T) {
	tests := []struct {
		name   string
		client ctrlruntimeclient.Client
		getter ManifestFSGetter
		want   []runtime.Object
	}{
		{
			name:   "test instantype - 1 OK- 1 invalid", // should return the instancetype kubermatic-standard-1, skipping the invalid one
			client: fakeclient,
			want:   getInstancetype(2, "8Gi", "standard-1"),
			getter: &testInstancetypeGetter{},
		},
		{
			name:   "test preference 1 OK- 1 invalid", // should return the preference kubermatic-standard-1, skipping the invalid one
			client: fakeclient,
			want:   getPreference("standard-1"),
			getter: &testPreferenceGetter{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RuntimeFromYaml(tt.client, tt.getter)
			if !reflect.DeepEqual(got, tt.want) {
				log(t, "RuntimeFromYaml() - Got:", got)
				log(t, "RuntimeFromYaml() - Want:", tt.want)
			}
		})
	}
}

func log(t *testing.T, text string, objs []runtime.Object) {
	t.Errorf("%s", text)
	for _, o := range objs {
		t.Errorf("%v ", o)
	}
}

func getInstancetype(cpu uint32, memory, name string) []runtime.Object {
	memoryQuantity, err := resource.ParseQuantity(memory)
	if err != nil {
		return nil
	}
	instancetype := &kvinstancetypev1alpha1.VirtualMachineInstancetype{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kvinstancetypev1alpha1.SchemeGroupVersion.String(),
			Kind:       "VirtualMachineInstancetype",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kvinstancetypev1alpha1.VirtualMachineInstancetypeSpec{
			CPU: kvinstancetypev1alpha1.CPUInstancetype{
				Guest: cpu,
			},
			Memory: kvinstancetypev1alpha1.MemoryInstancetype{
				Guest: memoryQuantity,
			},
		},
	}
	return []runtime.Object{instancetype}
}

func getPreference(name string) []runtime.Object {
	preference := &kvinstancetypev1alpha1.VirtualMachinePreference{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kvinstancetypev1alpha1.SchemeGroupVersion.String(),
			Kind:       "VirtualMachinePreference",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kvinstancetypev1alpha1.VirtualMachinePreferenceSpec{
			CPU: &kvinstancetypev1alpha1.CPUPreferences{
				PreferredCPUTopology: kvinstancetypev1alpha1.PreferThreads,
			},
		},
	}
	return []runtime.Object{preference}
}
