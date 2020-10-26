package addon

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/util/yaml"

	"k8s.io/apimachinery/pkg/runtime"
)

func txtFuncMap(overwriteRegistry string) template.FuncMap {
	funcs := sprig.TxtFuncMap()
	funcs["Registry"] = func(registry string) string {
		if overwriteRegistry != "" {
			return overwriteRegistry
		}
		return registry
	}

	return funcs
}

type TemplateData struct {
	Addon             *kubermaticv1.Addon
	Kubeconfig        string
	MajorMinorVersion string
	Cluster           *kubermaticv1.Cluster
	Credentials       resources.Credentials
	Variables         map[string]interface{}
	DNSClusterIP      string
	DNSResolverIP     string
	ClusterCIDR       string
}

func ParseFromFolder(log *zap.SugaredLogger, overwriteRegistry string, manifestPath string, data *TemplateData) ([]runtime.RawExtension, error) {
	var allManifests []runtime.RawExtension

	infos, err := ioutil.ReadDir(manifestPath)
	if err != nil {
		return nil, err
	}

	for _, info := range infos {
		filename := path.Join(manifestPath, info.Name())
		infoLog := log.With("file", filename)

		if info.IsDir() {
			infoLog.Debug("Found directory in manifest path. Ignoring.")
			continue
		}

		infoLog.Debug("Processing file")

		fbytes, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %v", filename, err)
		}

		tpl, err := template.New(info.Name()).Funcs(txtFuncMap(overwriteRegistry)).Parse(string(fbytes))
		if err != nil {
			return nil, fmt.Errorf("failed to parse file %s: %v", filename, err)
		}

		bufferAll := bytes.NewBuffer([]byte{})
		if err := tpl.Execute(bufferAll, data); err != nil {
			return nil, fmt.Errorf("failed to execute templating on file %s: %v", filename, err)
		}

		sd := strings.TrimSpace(bufferAll.String())
		if len(sd) == 0 {
			infoLog.Debug("Skipping file as its empty after parsing")
			continue
		}

		addonManifests, err := yaml.ParseMultipleDocuments(bufio.NewReader(bufferAll))
		if err != nil {
			return nil, fmt.Errorf("decoding failed for file %s: %v", filename, err)
		}
		allManifests = append(allManifests, addonManifests...)
	}

	return allManifests, nil
}
