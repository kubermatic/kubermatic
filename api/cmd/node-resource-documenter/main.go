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

	"github.com/Masterminds/sprig"
	"sigs.k8s.io/yaml"
)

// funcMap creates a function map needed for template execution.
func funcMap() template.FuncMap {
	funcs := sprig.TxtFuncMap()
	funcs["Registry"] = func(registry string) string {
		return registry
	}
	return funcs
}

// readYAML reads a YAML file and unmarshals it.
func readYAML(filepath string) ([]kv, error) {
	log.Printf("Parsing YAML file %q ...", filepath)
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	bs, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
		// Apply template.
	}
	t, err := template.
		New("addon").
		Funcs(funcMap()).
		Parse(string(bs))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = t.Execute(&buf, fillKV())
	if err != nil {
		return nil, err
	}
	// Unmarshal the contained YAMLs.
	blocks := strings.Split(buf.String(), "\n---\n")
	yamlKVs := []kv{}
	for _, block := range blocks {
		var yamlKV kv
		err = yaml.Unmarshal([]byte(block), &yamlKV)
		if err != nil {
			return nil, err
		}
		yamlKV.postprocess()
		kind := yamlKV.stringAt("kind")
		if kind == "Deployment" || kind == "DaemonSet" {
			// Only interested in deployments.
			yamlKVs = append(yamlKVs, yamlKV)
		}
	}
	return yamlKVs, nil
}

// yamlToDoc creates documentation out of the collected YAML.
func yamlToDoc(filepath string, yamlKV kv) *buffer {
	if len(yamlKV) == 0 {
		return nil
	}
	if !yamlKV.exists("spec", "template", "spec", "containers") {
		return nil
	}
	containers := yamlKV.kvAt("spec", "template", "spec", "containers")
	if !containers.any("resources") {
		return nil
	}

	doc := &buffer{}
	dir, filename := path.Split(filepath)
	dirs := strings.Split(dir, "/")
	addon := dirs[len(dirs)-2]

	doc.push("\n\n#### Addon: ")
	doc.push(addon)
	doc.push(" / File: ")
	doc.push(filename)

	containers.do(func(k string, v interface{}) {
		container := v.(kv)
		container.postprocess()

		if !container.exists("resources") {
			return
		}

		doc.push("\n\n##### Container: ", container.stringAt("name"), "\n")
		doc.push("\n```yaml\n")
		doc.push("limits:\n")
		doc.push("    cpu: ", container.stringAt("resources", "limits", "cpu"), "\n")
		doc.push("    memory: ", container.stringAt("resources", "limits", "memory"), "\n")
		doc.push("requests:\n")
		doc.push("    cpu: ", container.stringAt("resources", "requests", "cpu"), "\n")
		doc.push("    memory: ", container.stringAt("resources", "requests", "memory"), "\n")
		doc.push("```")
	})

	return doc
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
				yamlKVs, err := readYAML(filepath)
				if err != nil {
					return err
				}
				for _, yamlKV := range yamlKVs {
					doc.pushAll(yamlToDoc(filepath, yamlKV))
				}
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
