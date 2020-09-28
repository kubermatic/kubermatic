#!/usr/bin/env bash

# Copyright 2020 The Kubermatic Kubernetes Platform contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

### This README.md generator
###
### That generates README.md with all scripts from this directory described.

set -euo pipefail

cd "$(dirname "$0")/.."

README="hack/README.md"

cat <<EOF >"$README"
# Scripts list and their description

This file is generated

EOF

while IFS= read -r -d '' script_file; do
    help_text=$(awk -F '### ' '/^###/ {print $2}' "$script_file")
    if [[ -z "$help_text" ]]; then
        help_text="TBD"
    fi
    cat <<EOF >>"$README"
## ${script_file}

${help_text}

EOF
done < <(find hack/ -name '*.sh' -not -path 'hack/images/*' -print0 | sort --zero-terminated)
