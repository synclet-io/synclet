#!/usr/bin/env bash
# Writes the Artifact Hub annotations that depend on the release into the chart's Chart.yaml:
# artifacthub.io/changes from the chart version's CHANGELOG.md section and
# artifacthub.io/images from appVersion. Requires mikefarah yq v4.
# Usage: artifacthub-annotations.sh [chart-dir]
set -euo pipefail

chart="${1:-$(dirname "$0")/synclet}"
chart_yaml="$chart/Chart.yaml"
changelog="$chart/CHANGELOG.md"

version=$(yq '.version' "$chart_yaml")
app_version=$(yq '.appVersion' "$chart_yaml")

changes=$(awk -v version="$version" '
  function yamlstr(s) {
    gsub(/\\/, "\\\\", s)
    gsub(/"/, "\\\"", s)
    return "\"" s "\""
  }
  /^## / {
    if (in_section) exit
    if (index($0, "## [" version "]") == 1 || index($0, "## " version " ") == 1) in_section = 1
    next
  }
  !in_section { next }
  /^### / {
    heading = substr($0, 5)
    kind = heading == "Features" ? "added" : heading == "Bug Fixes" ? "fixed" : "changed"
    next
  }
  /^\* / {
    text = substr($0, 3)
    links = ""
    while (match(text, / ?\(\[[^]]+\]\([^)]+\)\)$/)) {
      link = substr(text, RSTART, RLENGTH)
      text = substr(text, 1, RSTART - 1)
      sub(/^ ?\(\[/, "", link)
      split(link, parts, /\]\(/)
      name = parts[1]
      url = parts[2]
      sub(/\)\)$/, "", url)
      label = name ~ /^#[0-9]+$/ ? "PR " name : "Commit " name
      links = "    - name: " yamlstr(label) "\n      url: " yamlstr(url) "\n" links
    }
    gsub(/\*\*/, "", text)
    printf "- kind: %s\n  description: %s\n", kind, yamlstr(text)
    if (links != "") printf "  links:\n%s", links
  }
' "$changelog")

if [ -z "$changes" ]; then
  echo "no CHANGELOG entries for version $version in $changelog" >&2
  exit 1
fi

images="- name: synclet
  image: ghcr.io/synclet-io/synclet:$app_version"

CHANGES="$changes" IMAGES="$images" yq -i '
  .annotations["artifacthub.io/changes"] = strenv(CHANGES) |
  .annotations["artifacthub.io/images"] = strenv(IMAGES)
' "$chart_yaml"
