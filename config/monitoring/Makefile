.PHONY: prometheus
prometheus: build-rules check-rules

build-rules:
	rm -f prometheus/rules/*.yaml
	./prometheus/rules/src/compile.sh

check-rules:
	promtool check rules prometheus/rules/*.yaml

.PHONY: format-dashboards
format-dashboards:
	find grafana/dashboards -name '*.json' -exec sh -c "jq --sort-keys '.' {} > {}.tmp && mv {}.tmp {}" \;
