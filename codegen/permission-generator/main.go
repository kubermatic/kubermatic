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

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"go.uber.org/zap"

	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
)

func main() {
	log := createLogger()
	if err := run(log); err != nil {
		log.Errorln(err)
		if errors.Is(err, InvalidConfigError{}) {
			printHelp(os.Stderr)
		}
		os.Exit(1)
	}
}

func run(log *zap.SugaredLogger) error {
	config := &Config{}
	config.FromFlags()
	flag.Parse()
	if err := config.InitAndValidate(); err != nil {
		return errors.Join(InvalidConfigError{}, err)
	}

	invoc, err := SearchFuncInvocationsForPackages(log, config.GoModDir, config.Pkgs, config.Filter)
	if err != nil {
		return err
	}

	// if we only want to print the funcs, exit early
	if config.PrintFuncs {
		PrintFuncInvocations(os.Stdout, invoc)
		return nil
	}

	policy, err := config.PoC.GeneratePolicy(invoc)
	if err != nil {
		return err
	}

	fmt.Println(string(policy))

	return nil
}

func createLogger() *zap.SugaredLogger {
	logOpts := kubermaticlog.NewDefaultOptions()
	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	return rawLog.Sugar()
}

func printHelp(w io.Writer) {
	fmt.Fprintf(w, `
Usage: permission-generator [options] <pkgs> <filter>

	Pkgs is a comma separated list of packages which should be searched (e.g "k8c.io/kubermatic/v2/pkg/provider/cloud/aws,k8c.io/machine-controller/pkg/cloudprovider/provider/aws").
	Packages must be fully qualified go module names.

	Search string can be any valid Go Regex (e.g. "github.com/aws/aws-sdk-go-v2/*")

Full Examples:
	permission-generator --provider=aws "github.com/mypackage,github.com/myotherpackage" "github.com/aws/aws-sdk-go-v2/*"

options:
	--provider	cloud provider to use (e.g. aws)
	--goModDir	root directory of the go module. This should contain the go.mod file. Defaults to your current working directory
	--mapper		mapper yaml file to be used. Defaults to the provided mapper for the selected provider
`)
}
