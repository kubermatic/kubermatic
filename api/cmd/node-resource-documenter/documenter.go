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
	"log"
	"path"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// documenter parses one configuration document, searches for Deployment,
// DaemonSet, or StatefulSet, and creates the according documentation for
// that.
type documenter struct {
	path    string
	content []byte
	doc     *buffer
}

func newDocumenter(p string, c []byte) *documenter {
	return &documenter{
		path:    p,
		content: c,
		doc:     newBuffer(),
	}
}

func (d *documenter) scanAll() error {
	log.Printf("Documenting ...")
	manifests := bytes.Split(d.content, []byte("\n---\n"))
	for _, manifest := range manifests {
		if err := d.scanManifest(manifest); err != nil {
			return err
		}
	}
	return nil
}

func (d *documenter) scanManifest(manifest []byte) error {
	var u unstructured.Unstructured
	if err := yaml.Unmarshal(manifest, &u); err != nil {
		return err
	}
	switch u.GetKind() {
	case "Deployment":
		var ad appsv1.Deployment
		if err := yaml.Unmarshal(manifest, &ad); err != nil {
			return err
		}
		d.addSpec(ad.Spec.Template.Spec)
	case "DaemonSet":
		var ad appsv1.DaemonSet
		if err := yaml.Unmarshal(manifest, &ad); err != nil {
			return err
		}
		d.addSpec(ad.Spec.Template.Spec)
	case "StatefulSet":
		var as appsv1.StatefulSet
		if err := yaml.Unmarshal(manifest, &as); err != nil {
			return err
		}
		d.addSpec(as.Spec.Template.Spec)
	}
	return nil
}

func (d *documenter) addSpec(ps corev1.PodSpec) {
	qf := func(q *resource.Quantity) string {
		s := q.String()
		if s == "0" {
			return "none"
		}
		return "\"" + s + "\""
	}
	dir, filename := path.Split(d.path)
	dirs := strings.Split(dir, "/")
	addon := dirs[len(dirs)-2]
	hasHeader := false
	// Iterate over the containers.
	for _, container := range ps.Containers {
		if container.Resources.Size() == 0 {
			continue
		}
		if !hasHeader {
			d.doc.push("\n\n#### Addon: ")
			d.doc.push(addon)
			d.doc.push(" / File: ")
			d.doc.push(filename)

			hasHeader = true
		}
		limitsCPU := container.Resources.Limits.Cpu()
		limitsMem := container.Resources.Limits.Memory()
		requestsCPU := container.Resources.Requests.Cpu()
		requestsMem := container.Resources.Requests.Memory()

		d.doc.push("\n\n##### Container: ", container.Name, "\n")
		d.doc.push("\n```yaml\n")
		d.doc.push("limits:\n")
		d.doc.push("  cpu: ", qf(limitsCPU), "\n")
		d.doc.push("  memory: ", qf(limitsMem), "\n")
		d.doc.push("requests:\n")
		d.doc.push("  cpu: ", qf(requestsCPU), "\n")
		d.doc.push("  memory: ", qf(requestsMem), "\n")
		d.doc.push("```")
	}
}

func (d *documenter) document() *buffer {
	return d.doc
}
