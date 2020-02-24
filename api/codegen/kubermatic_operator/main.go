// This application generates some manifests in the Kubermatic Helm chart
// based on the canonical source of truth, the Kubermatic Operator package.

package main

import (
	"io/ioutil"
	"log"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/common"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
)

func main() {
	writeYAML(common.DefaultBackupStoreContainer, "config/kubermatic/static/store-container.yaml")
	writeYAML(common.DefaultBackupCleanupContainer, "config/kubermatic/static/cleanup-container.yaml")
	writeYAML(common.DefaultKubernetesAddons, "config/kubermatic/static/master/kubernetes-addons.yaml")
	writeYAML(common.DefaultOpenshiftAddons, "config/kubermatic/static/master/openshift-addons.yaml")
	writeJSON(common.DefaultUIConfig, "config/kubermatic/static/master/ui-config.json")

	markup, err := yaml.Marshal(map[string]interface{}{
		"addons": common.DefaultAccessibleAddons,
	})
	if err != nil {
		log.Fatalf("Failed to encode accessible addons as YAML: %v", err)
	}

	writeYAML(string(markup), "config/kubermatic/static/master/accessible-addons.yaml")

	versionCfg := &operatorv1alpha1.KubermaticVersionsConfiguration{
		Kubernetes: common.DefaultKubernetesVersioning,
		Openshift:  common.DefaultOpenshiftVersioning,
	}

	versionsYAML, err := common.CreateVersionsYAML(versionCfg)
	if err != nil {
		log.Fatalf("Failed to encode versions as YAML: %v", err)
	}

	writeYAML(versionsYAML, "config/kubermatic/static/master/versions.yaml")

	updatesYAML, err := common.CreateUpdatesYAML(versionCfg)
	if err != nil {
		log.Fatalf("Failed to encode updates as YAML: %v", err)
	}

	writeYAML(updatesYAML, "config/kubermatic/static/master/updates.yaml")
}

func writeYAML(container string, filename string) {
	log.Printf("Generating %s...", filename)

	container = strings.Join([]string{
		"# This file has been generated using hack/update-kubermatic-chart.sh, do not edit.",
		"",
		strings.TrimSpace(container),
		"",
	}, "\n")

	if err := ioutil.WriteFile(filename, []byte(container), 0664); err != nil {
		log.Fatalf("Failed to write: %v", err)
	}
}

func writeJSON(data string, filename string) {
	log.Printf("Generating %s...", filename)

	data = strings.TrimSpace(data) + "\n"

	if err := ioutil.WriteFile(filename, []byte(data), 0664); err != nil {
		log.Fatalf("Failed to write: %v", err)
	}
}
