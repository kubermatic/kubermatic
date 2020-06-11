#!/usr/bin/env bash

set -euo pipefail

echo -e "\nUpdating fixtures...\n"
make test-update &>/dev/null
echo -e "\nUpdated fixtures, starting tests..."

make test || (echo -e "\n Failed to update fixtures! \n"; exit 1)

echo -e "\nSuccessfully updated fixtures! \n"
