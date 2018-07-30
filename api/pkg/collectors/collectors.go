package collectors

import (
	"github.com/prometheus/client_golang/prometheus"

	kubermaticinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"

	"k8s.io/client-go/informers"
)

// AvailableCollectors is a map of all available collectors
var AvailableCollectors = map[string]func(registry prometheus.Registerer, informerFactory informers.SharedInformerFactory, kubermaticInformerfactory kubermaticinformers.SharedInformerFactory){
	"clusters": MustRegisterClusterCollector,
}
