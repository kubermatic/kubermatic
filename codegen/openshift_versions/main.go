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

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
)

var (
	defaultOpenshiftVersions = []string{"4.1.9", "4.1.18"}
	components               = []string{
		"console",
		"hyperkube",
		"docker-builder",
		"deployer",
		"docker-registry",
		"coredns",
		"cli",
		"node",
		"multus-cni",
		"container-networking-plugins-supported",
		"container-networking-plugins-unsupported",
		"sriov-cni",
		"sriov-network-device-plugin",
		"ovn-kubernetes",
		"cluster-image-registry-operator",
		"hypershift",
		"cluster-dns-operator",
		"cluster-network-operator",
		"cloud-credential-operator"}
	codeTemplate = template.Must(template.New("base").Parse(templateRaw))
)

const templateRaw = `
package resources

import (
	"fmt"
)

{{- range $componentName, $componentTagMapping := .Components }}

func {{ $componentName }}Image(openshiftVersion, registry string)(string, error){
	switch openshiftVersion {
{{- range $openshiftVersion, $componentTag := $componentTagMapping }}
		case openshiftVersion{{ $openshiftVersion }}:
			return OpenshiftImageWithRegistry(openshiftImage + "@{{ $componentTag }}", openshiftVersion, "{{index $.ImageNames $componentName }}", registry)
{{- end }}
		default:
			return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}
{{- end -}}
`

func main() {
	generated, err := generateImageTagGetters(defaultOpenshiftVersions, osDockerResolver)
	if err != nil {
		log.Fatalf("error running codegen: %v", err)
	}
	if err := ioutil.WriteFile("zz_generated_image_tags.go", generated, 0644); err != nil {
		log.Fatalf("error writing generated file: %v", err)
	}
}

// osDockerResolver must run somewhere where docker credentials for the
// quay.io/openshift-release-dev/ocp-release image are configured.
func osDockerResolver(version string) (string, error) {
	// docker run --rm -it openshift/origin-cli oc adm release info quay.io/openshift-release-dev/ocp-release:$version
	cmd := exec.Command("docker",
		"run",
		"--rm",
		"openshift/origin-cli",
		"oc",
		"adm",
		"release",
		"info",
		"quay.io/openshift-release-dev/ocp-release:"+version)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed executing command %v: out:\n%s\nerr: %v",
			cmd.Args, string(out), err)
	}
	return string(out), nil
}

func generateImageTagGetters(openshiftVersions []string, resolver func(string) (string, error)) ([]byte, error) {
	// All tags contains a map of: componentName -> openshiftVerion -> DockerTag
	allTags := map[string]map[string]string{}
	imageNames := map[string]string{}
	for _, openshiftVersion := range openshiftVersions {
		rawOut, err := resolver(openshiftVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve openshift version %q: %v", openshiftVersion, err)
		}

		componentTagMapping, err := getTagsFromOut(rawOut)
		if err != nil {
			return nil, err
		}

		for component, tag := range componentTagMapping {
			camelCasedComponentName := strcase.ToLowerCamel(component)
			if _, ok := allTags[camelCasedComponentName]; !ok {
				allTags[camelCasedComponentName] = map[string]string{}
				imageNames[camelCasedComponentName] = component
			}
			allTags[camelCasedComponentName][sanitizeOpenshiftVersion(openshiftVersion)] = tag
		}
	}

	data := struct {
		Components map[string]map[string]string
		ImageNames map[string]string
	}{
		Components: allTags,
		ImageNames: imageNames,
	}

	buffer := bytes.NewBuffer([]byte{})
	if err := codeTemplate.Execute(buffer, data); err != nil {
		return nil, fmt.Errorf("failed to render template: %v", err)
	}

	return format.Source(buffer.Bytes())
}

func sanitizeOpenshiftVersion(in string) string {
	return strings.ReplaceAll(in, ".", "")
}

func getTagsFromOut(out string) (map[string]string, error) {
	componentTagMap := map[string]string{}
	for _, component := range components {
		for _, line := range strings.Split(out, "\n") {
			if !strings.Contains(line, component) {
				continue
			}
			fields := strings.Fields(strings.TrimSpace(line))
			if n := len(fields); n != 2 {
				return nil, fmt.Errorf("expected two string fields, but string %q contained %d", line, n)
			}
			if fields[0] != component {
				continue
			}
			componentTagMap[component] = fields[1]
		}
	}

	return componentTagMap, nil
}
