#!/usr/bin/env bash
set -xeuo pipefail

# Following best practices from https://www.elastic.co/guide/en/elasticsearch/reference/current/rolling-upgrades.html

# Disable shard allocation
curl --retry 10 --retry-delay 1 -X PUT "${POD_IP}:9200/_cluster/settings" -H 'Content-Type: application/json' -d'
{
  "persistent": {
    "cluster.routing.allocation.enable": "none"
  }
}
'

# Execute a synced flush
curl --retry 10 --retry-delay 1 -X POST "${POD_IP}:9200/_flush/synced"
