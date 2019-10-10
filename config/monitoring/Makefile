default: prometheus grafana

prometheus: build-rules check-rules

build-rules:
	rm -f prometheus/rules/*.yaml
	./prometheus/rules/src/compile.sh

check-rules:
	promtool check rules prometheus/rules/*.yaml

grafana: format-dashboards package-dashboards

format-dashboards:
	./grafana/hack/cleanup-dashboards.sh

package-dashboards:
	./grafana/hack/package-dashboards.sh
