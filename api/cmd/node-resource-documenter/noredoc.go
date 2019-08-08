package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// parseYAML analysis a YAML file for limits. But read them
// as text files as they are also templates (non-valid YAML).
func parseYAML(filename string) error {
	log.Printf("Parsing YAML file %q ...", filename)
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	resourcesMode := false
	resourcePrefixes := []string{"limits:", "requests:", "memory:", "cpu:"}
	s := bufio.NewScanner(f)
scan:
	for s.Scan() {
		l := s.Text()
		t := strings.Trim(l, " \t")
		if t == "resources:" {
			resourcesMode = true
			continue
		}
		if resourcesMode {
			for _, prefix := range resourcePrefixes {
				if strings.HasPrefix(t, prefix) {
					log.Printf("Resource: %s", t)
					continue scan
				}
			}
			// TODO: Take care for comments.
			resourcesMode = false
		}
	}
	return s.Err()
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
	var kubermaticDir = flag.String("kubermaticdir", ".", "directory containing the kubermatic sources")

	flag.Parse()

	addonsDir := filepath.Join(*kubermaticDir, "addons")

	log.Printf("Generating Kubermatic node resource documentation ...")
	if err := traverseAddons(addonsDir); err != nil {
		log.Printf("Error: %v", err)
		os.Exit(-1)
	}
	log.Printf("Done!")
}
