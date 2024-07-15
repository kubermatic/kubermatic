/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/distribution/reference"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"

	"k8c.io/kubermatic/v2/pkg/util/yamled"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

var imageRemaps map[string]string

func main() {
	var dynamicImages []string
	pflag.StringArrayVar(&dynamicImages, "dynamic", []string{}, "Split this image to dynamically inject a version variable.")
	pflag.Parse()

	if err := calculateImageRemaps(dynamicImages); err != nil {
		log.Fatalf("Invalid --dynamic flags: %v", err)
	}

	docs, err := decodeReader(os.Stdin)
	if err != nil {
		log.Fatalf("Failed to load input documents from stdin: %v", err)
	}

	encoder := yaml.NewEncoder(os.Stdout)

	for _, document := range docs {
		document, err := processDocument(document)
		if err != nil {
			log.Fatalf("Failed to process document: %v", err)
		}

		data, err := document.MarshalYAML()
		if err != nil {
			log.Fatalf("Failed to encode output as YAML: %v", err)
		}

		if err := encoder.Encode(data); err != nil {
			log.Fatalf("Failed to encode output as YAML: %v", err)
		}
	}
}

func processDocument(doc *yamled.Document) (*yamled.Document, error) {
	kind, _ := doc.GetString(yamled.Path{"kind"})
	if kind != "Deployment" && kind != "DaemonSet" && kind != "StatefulSet" {
		return doc, nil
	}

	for _, containerKind := range []string{"containers", "initContainers"} {
		basePath := yamled.Path{"spec", "template", "spec", containerKind}

		containers, _ := doc.GetArray(basePath)

		for i, container := range containers {
			container, err := processContainer(container)
			if err != nil {
				return nil, fmt.Errorf("invalid container: %w", err)
			}

			doc.Set(basePath.Append(i), container)
		}
	}

	return doc, nil
}

func processContainer(container any) (any, error) {
	cmap, ok := container.(map[string]any)
	if !ok {
		return nil, errors.New("container is not a map")
	}

	image, ok := cmap["image"]
	if !ok {
		return container, nil
	}

	imageStr, ok := image.(string)
	if !ok {
		return nil, errors.New("container image is not a string")
	}

	processed, err := processImage(imageStr)
	if !ok {
		return nil, err
	}

	cmap["image"] = processed

	return cmap, nil
}

func processImage(image string) (string, error) {
	tagged, err := parseImage(image)
	if err != nil {
		return "", err
	}

	remap, ok := imageRemaps[tagged.Name()]
	if ok {
		return fmt.Sprintf(`{{ Image (print "%s:" $%s) }}`, tagged.Name(), remap), nil
	}

	return fmt.Sprintf(`{{ Image "%s" }}`, image), nil
}

const (
	// 5 MB, same as chunk size in decoder
	bufSize = 5 * 1024 * 1024
)

func decodeReader(source io.ReadCloser) ([]*yamled.Document, error) {
	docSplitter := yamlutil.NewDocumentDecoder(source)
	defer docSplitter.Close()

	result := []*yamled.Document{}

	for i := 1; true; i++ {
		buf := make([]byte, bufSize)
		read, err := docSplitter.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return nil, fmt.Errorf("document %d is larger than the internal buffer", i)
		}

		doc, err := yamled.Load(bytes.NewReader(buf[:read]))
		if err != nil {
			return nil, fmt.Errorf("document %d is invalid: %v", i, err)
		}

		result = append(result, doc)
	}

	return result, nil
}

func calculateImageRemaps(flags []string) error {
	imageRemaps = map[string]string{}

	for _, flag := range flags {
		tagged, err := parseImage(flag)
		if err != nil {
			return fmt.Errorf("%s is invalid: %w", flag, err)
		}

		imageRemaps[tagged.Name()] = tagged.Tag()
	}

	return nil
}

func parseImage(image string) (reference.NamedTagged, error) {
	named, err := reference.ParseDockerRef(image)
	if err != nil {
		return nil, fmt.Errorf("invalid reference %q: %w", image, err)
	}

	tagged, ok := named.(reference.NamedTagged)
	if !ok {
		return nil, errors.New("image is not tagged")
	}

	return tagged, nil
}
