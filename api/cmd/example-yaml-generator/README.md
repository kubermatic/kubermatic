# Example YAML Generator

This application uses Kubernetes' test-infra/genyaml package to create
nice, commented YAML files based on Go structs. We use this to document
the available options for Seed CRDs.

## Usage

Use the scripts in `hack/` to update the generated YAML files in `docs/`:

    ./hack/update-yaml-examples.sh

Note that this needs to patch your local sources to strip out the
`omitempty` tag, so make sure to only run this on a clean working copy.
