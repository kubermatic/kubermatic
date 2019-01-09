.PHONY: prometheus
prometheus: check-rules

check-rules:
	promtool check rules prometheus/rules/*.yaml

.PHONY: format-dashboards
format-dashboards:
	find grafana/dashboards -name '*.json' -exec sh -c "jq --sort-keys '.' {} > {}.tmp && mv {}.tmp {}" \;
