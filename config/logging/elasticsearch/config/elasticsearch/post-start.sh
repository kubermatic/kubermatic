#!/usr/bin/env bash
set -xeuo pipefail

# Following best practices from https://www.elastic.co/guide/en/elasticsearch/reference/current/rolling-upgrades.html

# Wait until the node joined the cluster again
until [[ "$(curl --silent --show-error --connect-timeout 1 -H 'Content-Type: application/json' -X GET http://${POD_IP}:9200/_cat/nodes | grep ${POD_IP})" ]];
do
  echo "Node has not joined the cluster"
  sleep 1
done

# Enable shard allocation
curl --retry 10 --retry-delay 1 -X PUT "${POD_IP}:9200/_cluster/settings" -H 'Content-Type: application/json' -d'
{
  "persistent": {
    "cluster.routing.allocation.enable": null
  }
}
'
