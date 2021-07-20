package docs

import _ "embed"

//go:embed zz_generated.kubermaticConfiguration.yaml
var ExampleKubermaticConfiguration string

//go:embed zz_generated.seed.yaml
var ExampleSeedConfiguration string
