#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
manifest="${script_dir}/customizations/plugin-compiler-ng.yaml"
org="${DHI_ORG:-tykio}"
destination="${DHI_DESTINATION:-${org}/dhi-busybox-plugin-compiler}"
name="plugin compiler ng toolchain"

[[ "${org}" =~ ^[a-z0-9][a-z0-9._-]*$ ]] || {
  echo "ERROR: invalid DHI_ORG: ${org}" >&2
  exit 1
}
[[ "${destination}" =~ ^[a-z0-9][a-z0-9._-]*/[a-z0-9][a-z0-9._-]*$ ]] || {
  echo "ERROR: invalid DHI_DESTINATION: ${destination}" >&2
  exit 1
}

command -v docker >/dev/null 2>&1 || {
  echo "ERROR: docker is required" >&2
  exit 1
}
command -v jq >/dev/null 2>&1 || {
  echo "ERROR: jq is required" >&2
  exit 1
}
command -v go >/dev/null 2>&1 || {
  echo "ERROR: go is required" >&2
  exit 1
}

docker dhi --help >/dev/null

desired_manifest="$(mktemp)"
remote_manifest="$(mktemp)"
edit_manifest="$(mktemp)"
trap 'rm -f "${desired_manifest}" "${remote_manifest}" "${edit_manifest}"' EXIT

sed \
  "s#destination: tykio/dhi-busybox-plugin-compiler#destination: ${destination}#" \
  "${manifest}" >"${desired_manifest}"

customizations="$(docker dhi customization list \
  --org "${org}" \
  --repo "${destination}" \
  --filter "${name}" \
  --json)"

matches="$(
  jq -c --arg name "${name}" --arg repository "${destination}" '
    (if type == "array" then . else (.customizations // .items // []) end)
    | [.[] | select(
        (.name // "") == $name and
        (.repository // "") == $repository
      )]
  ' <<<"${customizations}"
)"
match_count="$(jq -r 'length' <<<"${matches}")"

if ((match_count > 1)); then
  echo "ERROR: multiple matching DHI customizations found for ${destination}" >&2
  jq -r '.[].id | "  \(.)"' <<<"${matches}" >&2
  exit 1
fi

if ((match_count == 0)); then
  docker dhi customization create "${desired_manifest}" --org "${org}"
  exit
fi

id="$(jq -r '.[0].id' <<<"${matches}")"
docker dhi customization get "${id}" --org "${org}" >"${remote_manifest}"

if ! comparison="$(
  cd "${repo_root}"
  go run ./dhi/cmd/customization-equal "${desired_manifest}" "${remote_manifest}"
)"; then
  echo "ERROR: failed to compare local and remote customizations" >&2
  exit 1
fi
case "${comparison}" in
  equal)
    echo "Customization '${name}' (ID: ${id}) is already up to date"
    exit
    ;;
  different) ;;
  *)
    echo "ERROR: unexpected customization comparison result: ${comparison}" >&2
    exit 1
    ;;
esac

{
  printf 'id: %s\n' "${id}"
  sed '/^id:/d' "${desired_manifest}"
} >"${edit_manifest}"

docker dhi customization edit "${edit_manifest}" --org "${org}"
