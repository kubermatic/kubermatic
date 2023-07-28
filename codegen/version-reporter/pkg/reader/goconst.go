/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package reader

import (
	"fmt"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

func ReadGoConstantFromPackage(dir string, pkgToSearch string, constName string) (string, error) {
	config := &packages.Config{
		Dir:        dir,
		BuildFlags: []string{"-tags=ee"},
		Mode:       packages.NeedTypes | packages.NeedTypesInfo,
	}

	pkgs, err := packages.Load(config, pkgToSearch)
	if err != nil {
		return "", err
	}

	sourcePkg := pkgs[0]

	// handle errors like no 'Go files found in pkg' upfront
	for _, err := range sourcePkg.Errors {
		return "", fmt.Errorf("failed to load package %q: %s", sourcePkg.ID, err.Msg) // we have to use sourcePkg.ID here, as fields like name are not set
	}

	for _, v := range sourcePkg.TypesInfo.Defs {
		if v == nil || !isPackageScoped(v) {
			continue
		}

		if constDef, constOk := v.(*types.Const); constOk && constDef.Name() == constName {
			quotedValue := constDef.Val().ExactString()

			return strconv.Unquote(quotedValue)
		}
	}

	return "", ErrVersionNotFound
}

func isPackageScoped(obj types.Object) bool {
	parent := obj.Parent()

	return parent != nil && strings.HasPrefix(parent.String(), "package ")
}
