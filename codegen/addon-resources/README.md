# Addon Resource Documenter

## Intention

The Addon Resource Documenter collects the configured resources of the Kubermatic
addons. This data is combined into a JSON file, which can then be used by the KKP
documentation to render a pretty Markdown file.

## Parameters

- `kubermaticdir` sets the directory containing the KKP sources, default is `.`
- `output` sets the output filename, by default `addonresources.json`.
