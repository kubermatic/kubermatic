#!/usr/bin/env bash

set -xeuo pipefail

API_BASE_URL=http://127.0.0.1:5601/api

## Wait until Kibana becomes ready.
#until [[ "$(curl --silent --show-error --connect-timeout 1 "${API_BASE_URL}/status" | grep '"green"')" ]]; do
#  echo "Kibana is not yet available or unhealthy."
#  sleep 1
#done
#
## Check if the logstash index pattern exists already.
#resp=$(curl -s "${API_BASE_URL}/saved_objects/_find?type=index-pattern&search_fields=title&search=logstash-")
#
#if [[ "$resp" =~ '"total":0' ]]; then
#  # Create index pattern
#  curl -X POST -H 'Content-Type: application/json' -H 'kbn-xsrf: true' "${API_BASE_URL}/saved_objects/index-pattern" -d'
#{
#  "attributes": {
#    "title": "logstash-*",
#    "timeFieldName": "@timestamp"
#  }
#}
#'
#fi
