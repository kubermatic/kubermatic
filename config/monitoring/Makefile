.PHONY: prometheus
prometheus: rules check

.PHONY: rules
rules:
	rm -rf prometheus/rules/*
	jsonnet -J vendor -m prometheus/rules -e '(import "prometheus/jsonnet/prometheus.jsonnet").prometheus' | xargs -I{} sh -c 'cat $$1 | gojsontoyaml > $$1.yaml; rm -f $$1' -- {}

.PHONY: check-config check-rules
check: check-config check-rules

check-config:
	promtool check config prometheus/config/prometheus.yml

check-rules:
	promtool check rules prometheus/rules/rules.yaml

.PHONY: prometheus-customer
prometheus-customer:
	mkdir -p prometheus/rules-customer
	jsonnet -J vendor -m prometheus/rules-customer -e '(import "prometheus/jsonnet/prometheus-customer.jsonnet").prometheus' | xargs -I{} sh -c 'cat $$1 | gojsontoyaml > $$1.yaml; rm -f $$1' -- {}
	echo -e "package prometheus\n\nconst prometheusRules = \`\n`cat prometheus/rules-customer/rules.yaml`\n\`" > ../../api/pkg/resources/prometheus/configmap-rules.go
	rm -rf prometheus/rules-customer
