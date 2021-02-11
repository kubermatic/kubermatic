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

package pprof

import (
	"context"
	"flag"
	"net/http"
	"net/http/pprof"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ manager.Runnable = &Opts{}
var _ manager.LeaderElectionRunnable = &Opts{}

type Opts struct {
	ListenAddress string
}

func (opts *Opts) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&opts.ListenAddress, "pprof-listen-address", ":6600", "The listen address for pprof. Set to `0` to disable it.")
}

func (opts *Opts) Start(ctx context.Context) error {
	if opts.ListenAddress == "" || opts.ListenAddress == "0" {
		<-ctx.Done()
		return nil
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return http.ListenAndServe(opts.ListenAddress, mux)
}

func (opts *Opts) NeedLeaderElection() bool {
	return false
}
