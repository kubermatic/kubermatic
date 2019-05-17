#!/usr/bin/env sh

cd $(dirname $0)

for dashboard in */*.json; do
  echo "$dashboard"

  tmpfile="$dashboard.tmp"

  cat "$dashboard" | \
    jq '(.templating.list[] | select(.type=="query") | .options) = []' | \
    jq '(.templating.list[] | select(.type=="query") | .refresh) = 2' | \
    jq '(.templating.list[] | select(.type=="query") | .current) = {}' | \
    jq '(.templating.list[] | select(.type=="datasource") | .current) = {}' | \
    jq '(.templating.list[] | select(.type=="interval") | .current) = {}' | \
    jq '(.panels[] | select(.scopedVars!=null) | .scopedVars) = {}' | \
    jq '(.templating.list[] | select(.type=="datasource") | .hide) = "<< datasourceHide | toJson >>"' | \
    jq '(.annotations.list) = []' | \
    jq '(.links) = []' | \
    jq '(.refresh) = "<< refresh | default `30s` | toJson >>"' | \
    jq '(.time.from) = "<< defaultRange | toJson >>"' | \
    jq '(.editable) = "<< editable | default false | toJson >>"' | \
    jq '(.panels[] | select(.type!="row") | .editable) = "<< editable | default false | toJson >>"' | \
    jq '(.panels[] | select(.type!="row") | .transparent) = "<< transparentPanels | default true | toJson >>"' | \
    jq '(.panels[] | select(.type!="row") | .timeRegions) = []' | \
    jq '(.hideControls) = "<< hideControls | default false | toJson >>"' | \
    jq '(.time.to) = "now"' | \
    jq '(.timezone) = ""' | \
    jq '(.graphTooltip) = 1' | \
    jq 'del(.panels[] | select(.repeatPanelId!=null))' | \
    jq 'del(.id)' | \
    jq 'del(.iteration)' | \
    jq --sort-keys '.' > "$tmpfile"

  mv "$tmpfile" "$dashboard"
done
