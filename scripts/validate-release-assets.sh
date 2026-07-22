#!/usr/bin/env bash

set -Eeuo pipefail

APPLICATION_NAME="proxmox-adguard-sync"
DEFAULT_REPOSITORY="ka0sdev/proxmox-adguard-sync"

usage() {
  cat <<'EOF'
Validate published GitHub Release assets.

Usage:
  ./scripts/validate-release-assets.sh <tag> [repository]

Examples:
  ./scripts/validate-release-assets.sh v0.1.0-beta.1
  ./scripts/validate-release-assets.sh v0.1.0-beta.1 ka0sdev/proxmox-adguard-sync

Requirements:
  gh
  sha256sum
  tar
  readelf
EOF
}

fail() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

require_command() {
  local command_name="$1"

  if ! command -v "${command_name}" >/dev/null 2>&1; then
    fail "required command not found: ${command_name}"
  fi
}

validate_archive_contents() {
  local archive="$1"
  local package_name="$2"

  local required_paths=(
    "${package_name}/${APPLICATION_NAME}"
    "${package_name}/README.md"
    "${package_name}/LICENSE"
    "${package_name}/.env.example"
  )

  local listing
  listing="$(tar -tzf "${archive}")"

  for required_path in "${required_paths[@]}"; do
    if ! grep -Fxq "${required_path}" <<<"${listing}"; then
      fail "$(basename "${archive}") is missing ${required_path}"
    fi
  done
}

validate_architecture() {
  local binary="$1"
  local expected_architecture="$2"

  local elf_header
  elf_header="$(readelf -h "${binary}")"

  case "${expected_architecture}" in
    amd64)
      if ! grep -Eq \
        'Machine:[[:space:]]+(Advanced Micro Devices X86-64|AMD x86-64)' \
        <<<"${elf_header}"; then
        printf '%s\n' "${elf_header}" >&2
        fail "$(basename "${binary}") is not an AMD64 binary"
      fi
      ;;

    arm64)
      if ! grep -Eq \
        'Machine:[[:space:]]+AArch64' \
        <<<"${elf_header}"; then
        printf '%s\n' "${elf_header}" >&2
        fail "$(basename "${binary}") is not an ARM64 binary"
      fi
      ;;

    *)
      fail "unsupported architecture check: ${expected_architecture}"
      ;;
  esac
}

validate_version() {
  local binary="$1"
  local expected_version="$2"

  local output
  output="$("${binary}" --version)"

  local expected_line="${APPLICATION_NAME} ${expected_version}"

  if ! grep -Fxq "${expected_line}" <<<"${output}"; then
    printf '%s\n' "${output}" >&2
    fail "binary version does not match ${expected_version}"
  fi

  if ! grep -Eq '^commit: [0-9a-f]{7,40}$' <<<"${output}"; then
    printf '%s\n' "${output}" >&2
    fail "binary does not contain valid commit metadata"
  fi

  if ! grep -Eq \
    '^built: [0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$' \
    <<<"${output}"; then
    printf '%s\n' "${output}" >&2
    fail "binary does not contain a valid UTC build timestamp"
  fi
}

if [[ $# -lt 1 || $# -gt 2 ]]; then
  usage
  exit 1
fi

TAG="$1"
REPOSITORY="${2:-${GITHUB_REPOSITORY:-${DEFAULT_REPOSITORY}}}"

if [[ ! "${TAG}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  fail "invalid release tag: ${TAG}"
fi

VERSION="${TAG#v}"

AMD64_PACKAGE="${APPLICATION_NAME}_${VERSION}_linux_amd64"
ARM64_PACKAGE="${APPLICATION_NAME}_${VERSION}_linux_arm64"

AMD64_ARCHIVE="${AMD64_PACKAGE}.tar.gz"
ARM64_ARCHIVE="${ARM64_PACKAGE}.tar.gz"
CHECKSUM_FILE="checksums.txt"

require_command gh
require_command sha256sum
require_command tar
require_command readelf

WORK_DIRECTORY="$(mktemp -d)"

cleanup() {
  rm -rf "${WORK_DIRECTORY}"
}

trap cleanup EXIT

DOWNLOAD_DIRECTORY="${WORK_DIRECTORY}/download"
EXTRACT_DIRECTORY="${WORK_DIRECTORY}/extract"

mkdir -p "${DOWNLOAD_DIRECTORY}" "${EXTRACT_DIRECTORY}"

printf 'Validating release %s from %s\n' "${TAG}" "${REPOSITORY}"
printf 'Downloading live release assets...\n'

gh release download "${TAG}" \
  --repo "${REPOSITORY}" \
  --dir "${DOWNLOAD_DIRECTORY}"

cd "${DOWNLOAD_DIRECTORY}"

required_assets=(
  "${AMD64_ARCHIVE}"
  "${ARM64_ARCHIVE}"
  "${CHECKSUM_FILE}"
)

for asset in "${required_assets[@]}"; do
  if [[ ! -f "${asset}" ]]; then
    fail "required release asset is missing: ${asset}"
  fi
done

printf '✓ All required assets present\n'

checksum_entries="$(
  awk 'NF >= 2 { print $2 }' "${CHECKSUM_FILE}" |
    sed -e 's/^\*//' -e 's|^\./||'
)"

for archive in "${AMD64_ARCHIVE}" "${ARM64_ARCHIVE}"; do
  if ! grep -Fxq "${archive}" <<<"${checksum_entries}"; then
    fail "${CHECKSUM_FILE} does not contain ${archive}"
  fi
done

sha256sum --check "${CHECKSUM_FILE}"

printf '✓ Checksums verified\n'

validate_archive_contents \
  "${AMD64_ARCHIVE}" \
  "${AMD64_PACKAGE}"

validate_archive_contents \
  "${ARM64_ARCHIVE}" \
  "${ARM64_PACKAGE}"

printf '✓ Archive contents verified\n'

tar -xzf "${AMD64_ARCHIVE}" \
  -C "${EXTRACT_DIRECTORY}"

tar -xzf "${ARM64_ARCHIVE}" \
  -C "${EXTRACT_DIRECTORY}"

AMD64_BINARY="${EXTRACT_DIRECTORY}/${AMD64_PACKAGE}/${APPLICATION_NAME}"
ARM64_BINARY="${EXTRACT_DIRECTORY}/${ARM64_PACKAGE}/${APPLICATION_NAME}"

if [[ ! -x "${AMD64_BINARY}" ]]; then
  fail "AMD64 binary is missing or not executable"
fi

if [[ ! -x "${ARM64_BINARY}" ]]; then
  fail "ARM64 binary is missing or not executable"
fi

validate_architecture "${AMD64_BINARY}" amd64
validate_architecture "${ARM64_BINARY}" arm64

printf '✓ Binary architectures validated\n'

case "$(uname -m)" in
  x86_64 | amd64)
    validate_version "${AMD64_BINARY}" "${VERSION}"
    ;;

  aarch64 | arm64)
    validate_version "${ARM64_BINARY}" "${VERSION}"
    ;;

  *)
    fail "cannot execute a release binary on host architecture $(uname -m)"
    ;;
esac

printf '✓ Version and build metadata verified\n'

printf '\n'
printf 'Release Asset Validation: PASSED\n'
printf 'Repository: %s\n' "${REPOSITORY}"
printf 'Tag: %s\n' "${TAG}"
printf 'Version: %s\n' "${VERSION}"
printf '\n'
printf 'Validation Summary\n'
printf '  All required assets present ✓\n'
printf '  Checksums verified ✓\n'
printf '  Archive contents verified ✓\n'
printf '  Version and build metadata verified ✓\n'
printf '  Binary architectures validated ✓\n'