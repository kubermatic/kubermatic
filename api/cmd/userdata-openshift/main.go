package main

import (
	"flag"

	"github.com/golang/glog"

	"github.com/kubermatic/kubermatic/api/pkg/userdata/openshift"
	"github.com/kubermatic/machine-controller/pkg/userdata/plugin"
)

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Switch for enabling the plugin debugging")
	flag.Parse()

	var provider = &openshift.Provider{}
	var p = plugin.New(provider, debug)

	if err := p.Run(); err != nil {
		glog.Fatalf("failed to run openshift userdata: %v", err)
	}
}
