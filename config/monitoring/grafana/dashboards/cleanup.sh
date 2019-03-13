#!/usr/bin/env sh

cd $(dirname $0)

for dashboard in */*.json; do
  echo "$dashboard"

  tmpfile="$dashboard.tmp"

  cat "$dashboard" | \
    jq '(.templating.list[] | select(.type=="query") | .options) = []' | \
    jq '(.templating.list[] | select(.type=="query") | .refresh) = 2' | \
    jq '(.templating.list[] | select(.type=="query") | .current) = {}' | \
    jq '(.annotations.list) = []' | \
    jq '(.links) = []' | \
    jq '(.refresh) = "<< refresh | default `30s` | toJson >>"' | \
    jq '(.time.from) = "<< defaultRange | toJson >>"' | \
    jq '(.time.to) = "now"' | \
    jq '(.timezone) = ""' | \
    jq '(.graphTooltip) = 1' | \
    jq --sort-keys '.' > "$tmpfile"

  mv "$tmpfile" "$dashboard"
done
