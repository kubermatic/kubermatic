#!/bin/bash
set -e

if [ -z "$REGION" ]; then
	echo "REGION is not set"
fi

curl -s --request GET "https://api.digitalocean.com/v2/droplets" \
  --header "Content-Type: application/json" --header "Authorization: Bearer $TOKEN" |
jq -r '.droplets[] | "\(.id) \(.region.slug)"' |
while read DROPLET_ID DROPLET_REGION; do
	if [ "$DROPLET_REGION" == "$REGION" ]; then
	    curl --request DELETE "https://api.digitalocean.com/v2/droplets/$DROPLET_ID" \
			--header "Content-Type: application/json" --header "Authorization: Bearer $TOKEN"
	fi
done