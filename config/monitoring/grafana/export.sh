#!/usr/bin/env bash

# Set KEYS from the outside passing a API key to this script:
# KEYS="XXX" bash export.sh

HOST="http://localhost:3000"
DIR=dashboards

fetch_fields() {
    echo $(curl -sSL -f -k -H "Authorization: Bearer ${1}" "${HOST}/api/${2}" | jq -r "if type==\"array\" then .[] else . end| .${3}")
}

for key in "${KEYS[@]}" ; do

    if [ ! -d "$DIR" ] ; then
        mkdir -p "$DIR"
    fi

    for dash in $(fetch_fields ${key} 'search?query=&' 'uri'); do
        DB=$(echo ${dash}|sed 's,db/,,g').json
        echo $DB
        curl -f -k -H "Authorization: Bearer ${key}" "${HOST}/api/dashboards/${dash}" | jq '.dashboard.id = null' | jq '.overwrite = true' > "$DIR/${DB%.json}-dashboard.json"
    done

    for id in $(fetch_fields ${key} 'datasources' 'id'); do
        DS=$(echo $(fetch_fields ${key} "datasources/${id}" 'name')|sed 's/ /-/g').json
        echo $DS
        curl -f -k -H "Authorization: Bearer ${key}" "${HOST}/api/datasources/${id}" | jq '.id = null' | jq '.orgId = null' > "$DIR/${DS%.json}-datasource.json"
    done

done
