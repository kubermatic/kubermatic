.PHONY: prometheus
prometheus: build-rules check-rules

build-rules:
	cd prometheus/rules/src; \
	for file in *.yaml; do echo $$file; yq r $$file -j | jq 'del(.groups[].rules[].runbook)' | yq r - > ../$$file; done

check-rules:
	promtool check rules prometheus/rules/*.yaml

.PHONY: format-dashboards
format-dashboards:
	find grafana/dashboards -name '*.json' -exec sh -c "jq --sort-keys '.' {} > {}.tmp && mv {}.tmp {}" \;
