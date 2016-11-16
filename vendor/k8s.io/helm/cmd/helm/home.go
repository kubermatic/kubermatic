/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

var longHomeHelp = `
This command displays the location of HELM_HOME. This is where
any helm configuration files live.
`

func newHomeCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "home",
		Short: "displays the location of HELM_HOME",
		Long:  longHomeHelp,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(out, homePath()+"\n")
		},
	}

	return cmd
}
