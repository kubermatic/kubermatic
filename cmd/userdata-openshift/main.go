package main

import (
	"flag"

	"github.com/kubermatic/kubermatic/api/pkg/userdata/openshift"
	"github.com/kubermatic/machine-controller/pkg/userdata/plugin"

	"k8s.io/klog"
)

func main() {
	klog.InitFlags(nil)
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Switch for enabling the plugin debugging")
	flag.Parse()

	var provider = &openshift.Provider{}
	var p = plugin.New(provider, debug)

	if err := p.Run(); err != nil {
		klog.Fatalf("failed to run openshift userdata: %v", err)
	}
}
