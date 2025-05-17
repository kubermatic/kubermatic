/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package applicationdefinitions

import (
	"embed"
	"io/fs"
)

//go:embed system-applications
var f embed.FS

func GetSysAppDefFiles() ([]fs.File, error) {
	dirname := "system-applications"
	files := []fs.File{}
	entries, err := f.ReadDir(dirname)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			file, err := f.Open(dirname + "/" + entry.Name())
			if err != nil {
				return nil, err
			}
			files = append(files, file)
		}
	}

	return files, nil
}
