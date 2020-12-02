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
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	"github.com/Masterminds/sprig/v3"
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

// readYAML reads a YAML file and executes the potential template.
func readYAML(path string) ([]byte, error) {
	log.Printf("Reading YAML file %q ...", path)
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	t, err := template.New("addon").Funcs(createFuncMap()).Parse(string(bs))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = t.Execute(&buf, createResources())
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// traverseAddons traverses the directories in kubermatic/addons
// to parse the individual found YAML FILES. It returns the generated
// documentation.
func traverseAddons(dir string) (*buffer, error) {
	// Prepare doc buffer.
	doc := newBuffer()
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
		func(path string, info os.FileInfo, err error) error {
			switch {
			case err != nil:
				return err
			case info.IsDir() || !strings.HasSuffix(info.Name(), ".yaml"):
				return nil
			default:
				c, err := readYAML(path)
				if err != nil {
					return err
				}
				d := newDocumenter(path, c)
				err = d.scanAll()
				if err != nil {
					return err
				}
				doc.pushAll(d.document())
			}
			return nil
		}); err != nil {
		return nil, err
	}
	doc.push("\n")
	return doc, nil
}

// writeDoc writes the documentation into the given file.
func writeDoc(file string, doc *buffer) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	return doc.writeAll(f)
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
