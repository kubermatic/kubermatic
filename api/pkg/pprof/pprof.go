package pprof

import (
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

func (opts *Opts) Start(stop <-chan struct{}) error {
	if opts.ListenAddress == "" || opts.ListenAddress == "0" {
		<-stop
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
