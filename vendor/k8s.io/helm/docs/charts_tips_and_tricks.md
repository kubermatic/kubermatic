# Chart Development Tips and Tricks

This guide covers some of the tips and tricks Helm chart developers have
learned while building production-quality charts.

## Know Your Template Functions

Helm uses [Go templates](https://godoc.org/text/template) for templating
your resource files. While Go ships several built-in functions, we have
added many others.

First, we added almost all of the functions in the
[Sprig library](https://godoc.org/github.com/Masterminds/sprig). We removed two
for security reasons: `env` and `expandenv` (which would have given chart authors
access to Tiller's environment).

We also added one special template function: `include`. The `include` function
allows you to bring in another template, and then pass the results to other
template functions.

For example, this template snippet includes a template called `mytpl.tpl`, then
lowercases the result, then wraps that in double quotes.

```yaml
value: {{include "mytpl.tpl" . | lower | quote}}
```

## Quote Strings, Don't Quote Integers

When you are working with string data, you are always safer quoting the
strings than leaving them as bare words:

```
name: {{.Values.MyName | quote }}
```

But when working with integers _do not quote the values._ That can, in
many cases, cause parsing errors inside of Kubernetes.

```
port: {{ .Values.Port }}
```

## Using the 'include' Function

Go provides a way of including one template in another using a built-in
`template` directive. However, the built-in function cannot be used in
Go template pipelines.

To make it possible to include a template, and then perform an operation
on that template's output, Helm has a special `include` function:

```
{{ include "toYaml" $value | indent 2 }}
```

The above includes a template called `toYaml`, passes it `$value`, and
then passes the output of that template to the `indent` function.

Because YAML ascribes significance to indentation levels and whitespace,
this is one great way to include snippets of code, but handle
indentation in a relevant context.

## Using "Partials" and Template Includes

Sometimes you want to create some reusable parts in your chart, whether
they're blocks or template partials. And often, it's cleaner to keep
these in their own files.

In the `templates/` directory, any file that begins with an
underscore(`_`) is not expected to output a Kubernetes manifest file. So
by convention, helper templates and partials are placed in a
`_helpers.tpl` file.

## Complex Charts with Many Dependencies

Many of the charts in the [official charts repository](https://github.com/kubernetes/charts)
are "building blocks" for creating more advanced applications. But charts may be
used to create instances of large-scale applications. In such cases, a single
umbrella chart may have multiple subcharts, each of which functions as a piece
of the whole.

The current best practice for composing a complex application from discrete parts
is to create a top-level umbrella chart that
exposes the global configurations, and then use the `charts/` subdirectory to
embed each of the components.

Two strong design patterns are illustrated by these projects:

**SAP's [OpenStack chart](https://github.com/sapcc/openstack-helm):** This chart
installs a full OpenStack IaaS on Kubernetes. All of the charts are collected
together in one GitHub repository.

**Deis's [Workflow](https://github.com/deis/workflow/tree/master/charts/workflow):**
This chart exposes the entire Deis PaaS system with one chart. But it's different
from the SAP chart in that this master chart is built from each component, and
each component is tracked in a different Git repository. Check out the
`requirements.yaml` file to see how this chart is composed by their CI/CD
pipeline.

Both of these charts illustrate proven techniques for standing up complex environments
using Helm.

## YAML is a Superset of JSON

According to the YAML specification, YAML is a superset of JSON. That
means that any valid JSON structure ought to be valid in YAML.

This has an advantage: Sometimes template developers may find it easier
to express a datastructure with a JSON-like syntax rather than deal with
YAML's whitespace sensitivity.

As a best practice, templates should follow a YAML-like syntax _unless_
the JSON syntax substantially reduces the risk of a formatting issue.

## Be Careful with Generating Random Values

There are functions in Helm that allow you to generate random data,
cryptographic keys, and so on. These are fine to use. But be aware that
during upgrades, templates are re-executed. When a template run
generates data that differs from the last run, that will trigger an
update of that resource.
