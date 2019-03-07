#!/usr/bin/env bash

set -xeuo pipefail

api=http://127.0.0.1:5601/api

# Wait until Kibana becomes ready.
until [[ "$(curl --silent --show-error --connect-timeout 1 $api/status | grep '"green"')" ]]; do
  echo "Kibana is not yet available or unhealthy."
  sleep 1
done

# Check if the logstash index pattern exists already.
resp=$(curl -s "$api/saved_objects/_find?type=index-pattern&search_fields=title&search=logstash-")

if [[ "$resp" =~ '"total":0' ]]; then
  # Create index pattern
  curl -X POST -H 'Content-Type: application/json' -H 'kbn-xsrf: true' "$api/saved_objects/index-pattern" -d'
{
  "attributes": {
    "title": "logstash-*",
    "timeFieldName": "@timestamp"
  }
}
'
fi
