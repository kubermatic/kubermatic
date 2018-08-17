# Grafana Helm Chart

This directory contains a Helm chart for deploying Grafana into
a Kubermatic seed cluster.

## Templating

To maintain a consistent style across all templates, the dashboard
files are not just plain JSON, but contain a couple of special
placeholders. A lot of the complexity comes from the fact that
Grafana and Go use the same delimiter for their placeholders and we
need to pay attention to when which placeholder is evaluated.

To allow creating new dashboards by just downloading a JSON file
straight from Grafana, the {{...}} placeholder is considered Grafana
stuff and kept as-is.
This means that our build-time placeholders have special forms.
To make sure that dashboard files are always valid JSON and can be
processed/formatted with tools like jq, the placeholder syntax uses
strings to enclose the placeholder. The examples below make this
concept easier to understand, don't worry.

  1. When you want to insert a *string* into a template, use the
     "[[ variable ]]" syntax, e.g.

     {
       "style": "[[ style ]]"
     }

     When you set `style: dark` in your values.yaml, this will
     result in a spectacularly expected output:

     {
       "style": "dark"
     }

  2. When you want to insert a number or boolean, you still have
     to wrap your placeholder in a string, but the string quotes
     will be removed during the evaluation.

     {
       "width": "<< width >>"
     }

     When you set `width: 42` in your values.yaml, this will
     result in this:

     {
       "style": 42
     }

     As you can see, the quotes are gone. You can use toJson to
     include even complex values:

     {
       "width": "<< width | toJson >>"
     }

     With `width: {left: 32, right: 34}` in your values.yml, this
     will result in:

     {
       "width": {"left": 32, "right": 34}
     }

The implementation inside the Helm template is somewhat ugly
thanks to Go's relatively limited templates, but given the above
knowledge, it should make sense.

There is one more thing to know: Originally, when accessing values
from the `values.yaml`, you would have to specify the full YAML path:

    [[ .Values.grafana.dashboards.yourKeyHere ]]

To prevent repetitive paths, both placeholders above ([[ and <<)
will prefix your variable with ".Values.grafana.dashboards.":

    [[ yourKeyHere ]]

    becomes

    {{ .Values.grafana.dashboards.yourKeyHere }}
