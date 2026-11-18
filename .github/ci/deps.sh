#!/usr/bin/env bash
# Prints the modules pinned in deps.ini as <module>@<version>, one per line.
# The /vN suffix comes from the version's major.
set -euo pipefail

while IFS='= ' read -r mod ver; do
  [[ -z "$mod" || "$mod" == [\;\#\[]* ]] && continue
  major="${ver%%.*}"; major="${major#v}"
  (( major >= 2 )) && mod="$mod/v$major"
  echo "$mod@$ver"
done < "$(dirname "${BASH_SOURCE[0]}")/../../deps.ini"
