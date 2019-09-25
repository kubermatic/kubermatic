package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"github.com/Masterminds/sprig"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// createFuncMap creates a function map needed for template execution.
func createFuncMap() template.FuncMap {
	funcs := sprig.TxtFuncMap()
	funcs["Registry"] = func(registry string) string {
		return registry
	}
	return funcs
}

// templateResources normally is a larger struct. But for the templates
// here the contained dummy cluster is okay.
type templateResources struct {
	Variables    map[string]interface{}
	Cluster      *kubermaticv1.Cluster
	DNSClusterIP string
}

// createResources creates dummy resources for the template.
func createResources() templateResources {
	res := templateResources{
		Variables: make(map[string]interface{}),
		Cluster: &kubermaticv1.Cluster{
			Spec: kubermaticv1.ClusterSpec{
				ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
					Pods: kubermaticv1.NetworkRanges{
						CIDRBlocks: []string{
							"172.25.0.0/16",
						},
					},
				},
			},
		},
		DNSClusterIP: "1.2.3.4",
	}
	res.Variables["NodeAccessNetwork"] = "172.26.0.0/16"
	return res
}

// specToDoc creates documentation out of a pod spec.
func specToDoc(filepath string, ps corev1.PodSpec) *buffer {
	qf := func(q *resource.Quantity) string {
		s := q.String()
		if s == "0" {
			return "none"
		}
		return "\"" + s + "\""
	}
	doc := &buffer{}
	dir, filename := path.Split(filepath)
	dirs := strings.Split(dir, "/")
	addon := dirs[len(dirs)-2]
	hasHeader := false
	// Iterate over the containers.
	for _, container := range ps.Containers {
		if container.Resources.Size() == 0 {
			continue
		}
		if !hasHeader {
			doc.push("\n\n#### Addon: ")
			doc.push(addon)
			doc.push(" / File: ")
			doc.push(filename)

			hasHeader = true
		}
		limitsCPU := container.Resources.Limits.Cpu()
		limitsMem := container.Resources.Limits.Memory()
		requestsCPU := container.Resources.Requests.Cpu()
		requestsMem := container.Resources.Requests.Memory()

		doc.push("\n\n##### Container: ", container.Name, "\n")
		doc.push("\n```yaml\n")
		doc.push("limits:\n")
		doc.push("    cpu: ", qf(limitsCPU), "\n")
		doc.push("    memory: ", qf(limitsMem), "\n")
		doc.push("requests:\n")
		doc.push("    cpu: ", qf(requestsCPU), "\n")
		doc.push("    memory: ", qf(requestsMem), "\n")
		doc.push("```")
	}
	return doc
}

// readYAML reads a YAML file, unmarshals it, and creates the
// documentation buffer.
func readYAML(filepath string) (*buffer, error) {
	log.Printf("Parsing YAML file %q ...", filepath)
	bs, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	// Prepare template.
	t, err := template.New("addon").Funcs(createFuncMap()).Parse(string(bs))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = t.Execute(&buf, createResources())
	if err != nil {
		return nil, err
	}
	// Unmarshal the contained YAMLs.
	blocks := strings.Split(buf.String(), "\n---\n")
	doc := &buffer{}
	for _, block := range blocks {
		bs := []byte(block)
		var u unstructured.Unstructured
		err = yaml.Unmarshal(bs, &u)
		if err != nil {
			return nil, err
		}
		kind := u.GetKind()
		switch kind {
		case "Deployment":
			var d appsv1.Deployment
			err = yaml.Unmarshal(bs, &d)
			if err != nil {
				return nil, err
			}
			doc.pushAll(specToDoc(filepath, d.Spec.Template.Spec))
		case "DaemonSet":
			var d appsv1.DaemonSet
			err = yaml.Unmarshal(bs, &d)
			if err != nil {
				return nil, err
			}
			doc.pushAll(specToDoc(filepath, d.Spec.Template.Spec))
		case "StatefulSet":
			var s appsv1.StatefulSet
			err = yaml.Unmarshal(bs, &s)
			if err != nil {
				return nil, err
			}
			doc.pushAll(specToDoc(filepath, s.Spec.Template.Spec))
		}
	}
	return doc, nil
}

// traverseAddons traverses the directories in kubermatic/addons
// to parse the individual found YAML FILES. It returns the generated
// documentation.
func traverseAddons(dir string) (*buffer, error) {
	// Prepare doc buffer.
	doc := &buffer{}
	doc.push("+++\n")
	doc.push("title = \"Kubermatic Addons - Resource\"\n")
	doc.push("date = " + time.Now().Format(time.RFC3339) + "\n")
	doc.push("weight = 7\n")
	doc.push("pre = \"<b></b>\"\n")
	doc.push("+++\n\n")
	doc.push("### Kubermatic Addons - Resources")
	// Walk over directories.
	if err := filepath.Walk(
		dir,
		func(filepath string, info os.FileInfo, err error) error {
			switch {
			case err != nil:
				return err
			case info.IsDir() || !strings.HasSuffix(info.Name(), ".yaml"):
				return nil
			default:
				filedoc, err := readYAML(filepath)
				if err != nil {
					return err
				}
				doc.pushAll(filedoc)
			}
			return nil
		}); err != nil {
		return nil, err
	}
	doc.push("\n")
	return doc, nil
}

// writeDoc writes the documentation into the given file.
func writeDoc(filepath string, doc *buffer) error {
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()
	for {
		line, ok := doc.pop()
		if !ok {
			break
		}
		if _, err := f.WriteString(line); err != nil {
			return err
		}
	}
	return nil
}

// main function of the resource limit documentation generator.
func main() {
	var kubermaticDir = flag.String("kubermaticdir", ".", "directory containing the kubermatic sources")
	var resourceLimitFile = flag.String("output", "_resource-limits.en.md", "path and filename for documentation")

	flag.Parse()

	addonsDir := filepath.Join(*kubermaticDir, "addons")

	log.Printf("Generating Kubermatic node resource documentation ...")

	doc, err := traverseAddons(addonsDir)
	if err != nil {
		log.Printf("Error traversing addons: %v", err)
		os.Exit(-1)
	}
	err = writeDoc(*resourceLimitFile, doc)
	if err != nil {
		log.Printf("Error writing documentation: %v", err)
		os.Exit(-1)
	}

	log.Printf("Done!")
}
