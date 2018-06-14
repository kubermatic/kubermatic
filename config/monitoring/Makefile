.PHONY: jsonnet
jsonnet: prometheus-jsonnet

.PHONY: prometheus-jsonnet
prometheus-jsonnet:
	rm -rf prometheus-statefulset/rules/*
	jsonnet -J vendor -m prometheus-statefulset/rules -e '(import "prometheus-statefulset/jsonnet/prometheus.jsonnet").prometheus' | xargs -I{} sh -c 'cat $$1 | gojsontoyaml > $$1.yaml; rm -f $$1' -- {}
