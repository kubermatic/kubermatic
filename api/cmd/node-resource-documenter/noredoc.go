package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	// "gopkg.in/yaml.v2"
)

// parseYAML analysis a YAML file for limits.
func parseYAML(path string) error {
	log.Printf("Parsing YAML file %q ...", path)
	return nil
}

// traverseAddons traverses the directories in kubermatic/addons
// to parse the individual found YAML FILES.
func traverseAddons(dir string) error {
	return filepath.Walk(
		dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(info.Name(), ".yaml") {
				return nil
			}
			return parseYAML(path)
		})
}

// main function of the resource limit documentation generator.
func main() {
	log.Printf("Generating Kubermatic node resource documentation ...")
	// TODO Get dir as argument.
	if err := traverseAddons("."); err != nil {
		log.Printf("Error: %v", err)
		os.Exit(-1)
	}
	log.Printf("Done!")
}
