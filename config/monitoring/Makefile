.PHONY: jsonnet
jsonnet: prometheus-jsonnet

.PHONY: prometheus-jsonnet
prometheus-jsonnet:
	rm -rf prometheus/rules/*
	jsonnet -J vendor -m prometheus/rules -e '(import "prometheus/jsonnet/prometheus.jsonnet").prometheus' | xargs -I{} sh -c 'cat $$1 | gojsontoyaml > $$1.yaml; rm -f $$1' -- {}

.PHONY: prometheus-check
prometheus-check:
	promtool check config prometheus/config/prometheus.yml
	promtool check rules prometheus/rules/rules.yaml
