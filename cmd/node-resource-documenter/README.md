# Node Resource Documenter

## Intention

The Node Resource Documenter collects the configured resources of the Kubermatic
addons. Based on this data a file for the content management at https://github.com/kubermatic/docs
is generated.

## Parameters

- `kubermaticdir` sets the directory containing the kubermatic sources, default is `.`
- `output` sets the directory and filename for the generated documentation, default is
  `_resource-limits.en.md`

## Output

The generated output file is a markdown file containing the header for the Hugo CMS. Inside
of it are sections for the addons containing resource configurations. Those are listed per
container in form of a YAML snippet describing the configuration.
