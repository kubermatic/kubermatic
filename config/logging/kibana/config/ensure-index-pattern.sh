#!/bin/bash

set -euo pipefail

if [[ ${DEBUG+false} ]]; then
  set -x
fi

kibana() {
   curl --silent -H 'kbn-xsrf:anything' "$@"
}

ensure_index_pattern() {
   output=$(kibana --show-error --connect-timeout 1 "${KIBANA_URL}/api/status")
   if [ $? -ne 0 ] || [ "${output:0:1}" != "{" ]; then
      echo "Kibana is not yet available."
      return 1
   fi

   if [[ "$(echo "$output" | jq -r '.status.overall.state')" != "green" ]]; then
      echo "Kibana is still unhealthy."
      return 1
   fi

   # Check if the index pattern already exists.
   patterns=$(kibana "${KIBANA_URL}/api/saved_objects/_find?type=index-pattern&search_fields=title&search=logstash-\\*" | jq '.total')
   if [ "$patterns" -ge 1 ]; then
      return 0
   fi

   echo "Creating index-pattern..."
   response=$(kibana -XPOST -H 'Content-Type:application/json' "${KIBANA_URL}/api/saved_objects/index-pattern" -d'
     {
       "attributes": {
         "title": "logstash-*",
         "timeFieldName": "@timestamp"
       }
     }
   ')

   # determine index pattern ID
   id=$(echo "$response" | jq -r '.id')
   if [[ -z "$id" ]]; then
      echo "Error: Could not find index-pattern we just tried to create!"
      return 1
   fi

   echo "Setting pattern as default..."
   config=$(curl --silent "${ELASTICSEARCH_HOSTS}/.kibana/doc/config:${KIBANA_VERSION}" | jq '._source // {"type": "config","config":{}}')

   echo "$config" | \
      jq ".config.defaultIndex = \"${id}\"" | \
      curl --silent -XPUT -H 'Content-Type:application/json' "${ELASTICSEARCH_HOSTS}/.kibana/doc/config:$KIBANA_VERSION" -d @- > /dev/null

   echo "Kibana successfully configured."
}

# try endlessly to reconcile the configuration
while true; do
   if ensure_index_pattern; then
      sleep 10
   else
      sleep 1
   fi
done
