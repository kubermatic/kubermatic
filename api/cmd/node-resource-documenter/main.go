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

// kv simulates the nested structs for template execution.
type kv map[string]interface{}

// fillKV creates the key/values needed for template execution.
func fillKV() kv {
	return kv{
		"Cluster": kv{
			"Spec": kv{
				"ClusterNetwork": kv{
					"Pods": kv{
						"CIDRBlocks": []string{"first"},
					},
				},
			},
		},
	}
}

// funcMap creates a function map needed for template execution.
func funcMap() template.FuncMap {
	funcs := sprig.TxtFuncMap()
	funcs["Registry"] = func(registry string) string {
		return registry
	}
	return funcs
}

// readYAML reads a YAML file and unmarshals it.
func readYAML(filepath string) (kv, error) {
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
	// Unmarshal the YAML.
	var yamlKV kv
	err = yaml.Unmarshal(buf.Bytes(), &yamlKV)
	return yamlKV, err
}

// resourcesToDoc creates documentation out of the
// collected configuration.
func resourcesToDoc(filepath string, out *buffer) *buffer {
	doc := &buffer{}
	dir, filename := path.Split(filepath)
	isCode := false
	counter := 0
	dirs := strings.Split(dir, "/")
	addon := dirs[len(dirs)-2]

	doc.push("#### Addon: ")
	doc.push(addon)
	doc.push(" / File: ")
	doc.push(filename)
	doc.push("\n\n")

	for {
		line, ok := out.pop()
		if !ok {
			break
		}
		switch {
		case strings.HasPrefix(line, "container: "):
			name := line[11:]
			if isCode {
				if counter == 0 {
					doc.push("none\n")
				}
				doc.push("```\n\n")
				counter = 0
			}
			doc.push("##### Container: ")
			doc.push(name)
			doc.push("\n\n```yaml\n")
			isCode = true
		case line == "limits:" || line == "requests:":
			doc.push(line)
			doc.push("\n")
			counter++
		default:
			doc.push("    ")
			doc.push(line)
			doc.push("\n")
			counter++
		}
	}

	if counter == 0 {
		doc.push("none\n")
	}
	doc.push("```\n\n")

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
	doc.push("### Kubermatic Addons - Resources\n\n")
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
				_, err := readYAML(filepath)
				if err != nil {
					return err
				}
				// if !out.isEmpty() {
				//	doc.pushAll(resourcesToDoc(filepath, out))
				// }
			}
			return nil
		}); err != nil {
		return nil, err
	}
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
