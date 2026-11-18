#!/usr/bin/env bash
# Writes a CycloneDX SBOM of the modules pinned in deps.ini, i.e. what the plugin adds on top of CoreDNS.
# Usage: sbom.sh <out.json>
set -euo pipefail

OUT="${1:?usage: sbom.sh <out.json>}"

"$(dirname "${BASH_SOURCE[0]}")/deps.sh" |
  jq -Rn '{bomFormat: "CycloneDX", specVersion: "1.5", version: 1,
           components: [inputs | split("@") | {type: "library", name: .[0], version: .[1],
                                                purl: "pkg:golang/\(.[0])@\(.[1])"}]}' > "$OUT"
