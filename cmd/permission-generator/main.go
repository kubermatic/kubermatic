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

	return nil
}

func createLogger() *zap.SugaredLogger {
	logOpts := kubermaticlog.NewDefaultOptions()
	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	return rawLog.Sugar()
}

func policyForPackages(poc PolicyCreator, pkgsToSearch []string, filter string) ([]byte, error) {
	invoc, err := SearchFuncInvocationsForPackages(pkgsToSearch, filter)
	if err != nil {
		return nil, err
	}
	return poc.GeneratePolicy(invoc)
}

func printHelp(w io.Writer) {
	fmt.Fprintf(w, `
Usage: permission-generator [options] <pkgs> <filter>

	Pkgs is a comma separated list of packages which should be searched (e.g "k8c.io/kubermatic/v2/pkg/provider/cloud/aws,github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/aws").
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
