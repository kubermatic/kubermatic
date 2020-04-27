package kubernetesdashboard

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
)

// ServiceCreator creates the service for the dashboard-metrics-scraper
func ServiceCreator(cidrBlocks []string) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return resources.MetricsScraperServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Name = resources.MetricsScraperServiceName
			s.Labels = resources.BaseAppLabels(scraperName, nil)
			s.Spec.Selector = resources.BaseAppLabels(scraperName, nil)
			s.Spec.Ports = []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       8000,
					TargetPort: intstr.FromInt(8000),
				},
			}
			if s.Spec.ClusterIP == "" {
				clusterIP, err := getMetricsScraperServiceIP(cidrBlocks)
				if err != nil {
					return nil, err
				}
				s.Spec.ClusterIP = clusterIP
			}
			return s, nil
		}
	}
}

// To avoid IP collision with the kube-dns service(https://github.com/kubermatic/kubermatic/issues/5232), we ensure that the IP used for this service is from a range outside the expected IP for the kube-dns service.
func getMetricsScraperServiceIP(clusterCIDRBlocks []string) (string, error) {
	rand.Seed(time.Now().UnixNano())
	if len(clusterCIDRBlocks) == 0 {
		return "", errors.New("failed to get metrics scarapper service IP: no service cidr defined")
	}

	block := clusterCIDRBlocks[0]
	_, ipnet, err := net.ParseCIDR(block)
	if err != nil {
		return "", fmt.Errorf("failed to get metrics scarapper service IP: invalid service cidr %s", block)
	}
	ip := ipnet.IP

	ip[len(ip)-1] = ip[len(ip)-1] + byte((rand.Intn(200) + 20))
	if !ipnet.Contains(ip) { // highly unlikely, but checking anyway.
		return "", fmt.Errorf("failed to get a valid service IP")
	}

	return ip.String(), nil
}
