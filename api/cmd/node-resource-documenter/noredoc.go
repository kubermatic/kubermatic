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

// parser is a function type for the state machine analyzing
// the YAML lines (including the templates which make a YAML
// lib unusable).
type parser func(current parser, line string, lines []string) ([]string, parser)

// resourcesToDoc creates documentation out of the
// collected configuration.
func resourcesToDoc(filepath string, lines []string) []string {
	var resdoc []string

	dir, filename := path.Split(filepath)

	isCode := false
	dirs := strings.Split(dir, "/")
	addon := dirs[len(dirs)-2]
	resdoc = append(resdoc, "### Addon: ", addon, " / File: ", filename, "\n\n")

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "container: "):
			container := line[11:]
			if isCode {
				resdoc = append(resdoc, "\n\n```\n\n")
			}
			resdoc = append(resdoc, "#### Container: ", container, "\n\n")
			resdoc = append(resdoc, "```\n")
			isCode = true
		case line == "limits:" || line == "requests:":
			resdoc = append(resdoc, line, "\n")
		default:
			resdoc = append(resdoc, "\t", line, "\n")
		}
	}

	resdoc = append(resdoc, "```\n\n")

	return resdoc
}

// mkParseResourceValues creates a parser for the resource values.
func mkParseResourceValues(index int) parser {
	return func(current parser, line string, lines []string) ([]string, parser) {
		trimmed := strings.Trim(line, " \t")
		switch {
		case trimmed == "limits:" || trimmed == "requests:":
			return append(lines, trimmed), current
		case strings.HasPrefix(trimmed, "memory:") || strings.HasPrefix(trimmed, "cpu:"):
			return append(lines, trimmed), current
		case strings.HasPrefix(trimmed, "#"):
			return lines, current
		}
		// TODO Currently don't handle multiple containes in one list.
		return lines, parseContainer
	}
}

// mkParseResources creates a parser for the "resources:" field.
func mkParseResources(index int) parser {
	return func(current parser, line string, lines []string) ([]string, parser) {
		if strings.IndexFunc(line, func(r rune) bool {
			return r != ' ' && r != '\t'
		}) < index {
			return lines, mkParseContainerName(index)
		}
		if strings.Index(line, "resources:") == -1 {
			return lines, current
		}
		return lines, mkParseResourceValues(index)
	}
}

// mkParseContainerName creates a parser for the container name.
func mkParseContainerName(index int) parser {
	return func(current parser, line string, lines []string) ([]string, parser) {
		if strings.Index(line, "- name: ") != index {
			return lines, current
		}
		name := strings.Trim(line, " \t")[8:]
		return append(lines, "container: "+name), mkParseResources(index)
	}
}

// parseContainer parses until "containers:" is found.
func parseContainer(current parser, line string, lines []string) ([]string, parser) {
	ci := strings.Index(line, "containers:")
	if ci == -1 {
		return lines, current
	}
	return lines, mkParseContainerName(ci)
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

	lines := []string{}
	p := parseContainer
	s := bufio.NewScanner(f)

	for s.Scan() {
		line := s.Text()
		lines, p = p(p, line, lines)
	}
	return lines, s.Err()
}

// traverseAddons traverses the directories in kubermatic/addons
// to parse the individual found YAML FILES. It returns the generated
// documentation.
func traverseAddons(dir string) ([]string, error) {
	var doc []string
	if err := filepath.Walk(
		dir,
		func(filepath string, info os.FileInfo, err error) error {
			switch {
			case err != nil:
				return err
			case info.IsDir() || !strings.HasSuffix(info.Name(), ".yaml"):
				return nil
			default:
				lines, err := parseYAML(filepath)
				if err != nil {
					return err
				}
				if len(lines) > 1 {
					doc = append(doc, resourcesToDoc(filepath, lines)...)
				}
			}
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
