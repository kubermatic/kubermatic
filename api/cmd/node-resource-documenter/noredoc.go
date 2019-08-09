package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// resourcesToDoc creates documentation out of the
// collected configuration.
func resourcesToDoc(filepath, containername string, lines []string) []string {
	if len(lines) == 0 {
		return nil
	}

	var resdoc []string

	_, filename := path.Split(filepath)

	resdoc = append(resdoc, "### File: ", filename, "\n\n")
	resdoc = append(resdoc, "#### Container: ", containername, "\n\n")
	resdoc = append(resdoc, "```\n")

	for _, line := range lines {
		if line == "limits:" || line == "requests:" {
			resdoc = append(resdoc, line, "\n")
			continue
		}
		resdoc = append(resdoc, "\t", line, "\n")
	}

	resdoc = append(resdoc, "```\n\n")

	return resdoc
}

// parseYAML analysis a YAML file for limits. But read them
// as text files as they are also templates (non-valid YAML).
func parseYAML(filepath string) ([]string, error) {
	log.Printf("Parsing YAML file %q ...", filepath)
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	resourcesMode := false
	resourcePrefixes := []string{"limits:", "requests:", "memory:", "cpu:"}

	var filedoc []string
	var lines []string
	var containername string

	s := bufio.NewScanner(f)
scan:
	for s.Scan() {
		l := s.Text()
		t := strings.Trim(l, " \t")
		switch {
		case strings.HasPrefix(t, "- name: "):
			containername = t[8:]
			continue
		case t == "resources:":
			resourcesMode = true
			continue
		case resourcesMode:
			for _, prefix := range resourcePrefixes {
				if strings.HasPrefix(t, prefix) {
					lines = append(lines, t)
					continue scan
				}
			}
			// TODO: Take care for comments.
			resdoc := resourcesToDoc(filepath, containername, lines)
			filedoc = append(filedoc, resdoc...)
			containername = ""
			lines = []string{}
			resourcesMode = false
		}
	}
	return filedoc, s.Err()
}

// traverseAddons traverses the directories in kubermatic/addons
// to parse the individual found YAML FILES. It returns the generated
// documentation.
func traverseAddons(dir string) ([]string, error) {
	var doc []string
	if err := filepath.Walk(
		dir,
		func(filepath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(info.Name(), ".yaml") {
				return nil
			}
			filedoc, err := parseYAML(filepath)
			if err != nil {
				return err
			}
			doc = append(doc, filedoc...)
			return nil
		}); err != nil {
		return nil, err
	}
	return doc, nil
}

// writeDoc writes the documentation into the given file.
func writeDoc(filepath string, doc []string) error {
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, line := range doc {
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
