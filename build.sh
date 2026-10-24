#!/usr/bin/env bash
# Blacksand cross-platform release builder.
#
# The default mode builds one target and embeds the already-exported frontend.
# `./build.sh --release` builds and packages all supported desktop/server targets.
#
# Environment variables:
#   BLACKSAND_TARGET_OS=linux             One target OS in single-target mode.
#   BLACKSAND_TARGET_ARCH=amd64           One target arch in single-target mode.
#   BLACKSAND_TARGETS=linux/amd64,...     Comma-separated targets for multi-target mode.
#   BLACKSAND_BUILD_VERSION=v0.3.3        Version embedded in the binary and archive name.
#   BLACKSAND_OUTPUT=/path/to/blacksand       Explicit binary path in single-target mode.
#   BLACKSAND_OUTPUT_DIR=dist             Directory for default binary paths.
#   BLACKSAND_PACKAGE=1                   Create a zip archive for each target.
#   BLACKSAND_PACKAGE_DIR=dist            Directory for release archives.
#   BLACKSAND_COMPRESS=off                UPX mode: off, auto, or required.
#   BLACKSAND_UPX_ARGS="--best --lzma"    Arguments passed to UPX.
#   BLACKSAND_SKIP_FRONTEND=1             Reuse server/webui/dist (for CI artifact builds).
#   BLACKSAND_SKIP_NPM_CI=1               Skip npm ci while rebuilding the frontend.
#   BLACKSAND_GOSUMDB=sum.golang.org      Go checksum database.
set -euo pipefail


cd "$(cd "$(dirname "$0")" && pwd)"

info() { printf '\033[36m[*]\033[0m %s\n' "$*"; }
ok() { printf '\033[32m[+]\033[0m %s\n' "$*"; }
warn() { printf '\033[33m[!]\033[0m %s\n' "$*" >&2; }
die() { printf '\033[31m[x]\033[0m %s\n' "$*" >&2; exit 1; }

usage() {
    cat <<'EOF'
Usage: ./build.sh [--target OS/ARCH | --release] [--upx | --no-compress]
Default: build the current system and architecture.
--release: build and zip Linux/macOS amd64/arm64 and Windows amd64.
--target OS/ARCH: select a target, for example windows/amd64.
--upx: require UPX compression (may reduce compatibility on some Linux systems).
--no-compress: use Go linker stripping and zip compression without UPX.
--help: show help.
Override targets with BLACKSAND_TARGETS=linux/amd64,windows/amd64 ./build.sh --release.
EOF
}

RELEASE_TARGETS_DEFAULT="linux/amd64,linux/arm64,darwin/amd64,darwin/arm64,windows/amd64"
BLACKSAND_RELEASE="${BLACKSAND_RELEASE:-0}"
BLACKSAND_COMPRESS="${BLACKSAND_COMPRESS:-off}"
BLACKSAND_PACKAGE="${BLACKSAND_PACKAGE:-0}"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --release)
      BLACKSAND_RELEASE=1
      BLACKSAND_PACKAGE=1
      shift
      ;;
    --target)
      [ "$#" -ge 2 ] || die "--target requires OS/ARCH"
      target_arg="$2"
      case "$target_arg" in
        */*)
          BLACKSAND_TARGET_OS="${target_arg%%/*}"
          BLACKSAND_TARGET_ARCH="${target_arg##*/}"
          BLACKSAND_TARGETS="$target_arg"
          ;;
        *) die "Target must be OS/ARCH, e.g. linux/amd64" ;;
      esac
      shift 2
      ;;
    --no-compress)
      BLACKSAND_COMPRESS=0
      shift
      ;;
    --upx)
      BLACKSAND_COMPRESS=required
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *) die "Unknown argument: $1 (see --help)" ;;
  esac
done

command -v go >/dev/null 2>&1 || die "Go not found (Go 1.26 or later required)"

BLACKSAND_GOSUMDB="${BLACKSAND_GOSUMDB:-sum.golang.org}"
if [ -z "${BLACKSAND_BUILD_VERSION:-}" ]; then
  if command -v git >/dev/null 2>&1 && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    BLACKSAND_BUILD_VERSION="$(git describe --tags --always --dirty)"
  else
    BLACKSAND_BUILD_VERSION="dev"
  fi
fi
# Release tags are commonly passed as v0.3.3; keep the binary version consistent.
BLACKSAND_BUILD_VERSION="${BLACKSAND_BUILD_VERSION#v}"
BLACKSAND_OUTPUT_DIR="${BLACKSAND_OUTPUT_DIR:-dist}"
BLACKSAND_PACKAGE_DIR="${BLACKSAND_PACKAGE_DIR:-$BLACKSAND_OUTPUT_DIR}"
BLACKSAND_UPX_ARGS="${BLACKSAND_UPX_ARGS:---best --lzma}"

if [ "${BLACKSAND_RELEASE}" = "1" ]; then
  BLACKSAND_TARGETS="${BLACKSAND_TARGETS:-$RELEASE_TARGETS_DEFAULT}"
else
  BLACKSAND_TARGET_OS="${BLACKSAND_TARGET_OS:-$(GOSUMDB="$BLACKSAND_GOSUMDB" go env GOOS)}"
  BLACKSAND_TARGET_ARCH="${BLACKSAND_TARGET_ARCH:-$(GOSUMDB="$BLACKSAND_GOSUMDB" go env GOARCH)}"
  BLACKSAND_TARGETS="${BLACKSAND_TARGETS:-${BLACKSAND_TARGET_OS}/${BLACKSAND_TARGET_ARCH}}"
fi

if [ "${BLACKSAND_SKIP_FRONTEND:-0}" = "1" ]; then
  [ -d server/webui/dist ] || die "BLACKSAND_SKIP_FRONTEND=1 but server/webui/dist is missing"
else
  command -v npm >/dev/null 2>&1 || die "npm not found (Node.js/npm required for frontend export)"
  command -v rsync >/dev/null 2>&1 || die "rsync not found"
  info "Building frontend static assets"
  if [ "${BLACKSAND_SKIP_NPM_CI:-0}" != "1" ]; then
    (cd web && npm ci)
  fi
  (cd web && npm run build:static)
  info "Syncing frontend to server/webui/dist"
  mkdir -p server/webui/dist
  rsync -a --delete web/out/ server/webui/dist/
fi

compress_binary() {
  binary="$1"
  goos="$2"
  case "$BLACKSAND_COMPRESS" in
    0|off|false|none)
      info "Skipping UPX: $binary"
      return 0
      ;;
    auto|required|true|1) ;;
    *) die "BLACKSAND_COMPRESS must be off, auto, or required" ;;
  esac

  if ! command -v upx >/dev/null 2>&1; then
    if [ "$BLACKSAND_COMPRESS" = "required" ]; then
      die "BLACKSAND_COMPRESS=required but upx was not found"
    fi
    warn "upx not found; retaining linker output: $binary"
    return 0
  fi

  before=$(wc -c < "$binary" | tr -d ' ')
  upx_args="$BLACKSAND_UPX_ARGS"
  [ "$goos" = "darwin" ] && upx_args="$upx_args --force-macos"
  # shellcheck disable=SC2086
  if ! upx $upx_args -- "$binary"; then
    if [ "$BLACKSAND_COMPRESS" = "required" ]; then
      die "UPX compression failed: $binary"
    fi
    warn "UPX does not support this format; retaining binary: $binary"
    return 0
  fi
  after=$(wc -c < "$binary" | tr -d ' ')
  ok "UPX complete: $binary (${before} -> ${after} bytes)"
}

package_binary() {
  binary="$1"
  goos="$2"
  goarch="$3"
  package_name="blacksand-${BLACKSAND_BUILD_VERSION}-${goos}-${goarch}"
  package_root="${BLACKSAND_PACKAGE_DIR}/${package_name}"
  archive="${BLACKSAND_PACKAGE_DIR}/${package_name}.zip"

  command -v zip >/dev/null 2>&1 || die "Packaging requires zip"
  rm -rf "$package_root" "$archive"
  mkdir -p "$package_root"
  cp "$binary" "$package_root/"
  # Package the platform supervisor so updates can restart the process.
  if [ "$goos" = "windows" ]; then
    cp start.bat "$package_root/"
  else
    cp start.sh "$package_root/"
    chmod +x "$package_root/start.sh"
  fi
  cp -R skills "$package_root/"
  cp config.example.json "$package_root/"
  cp README.md LICENSE CHANGELOG.md CONTRIBUTING.md SECURITY.md SUPPORT.md CODE_OF_CONDUCT.md "$package_root/"
  mkdir -p "$package_root/notices"
  cp web/LICENSE "$package_root/notices/frontend-MIT.txt"
  if [ -d docs ]; then cp -R docs "$package_root/"; fi
  if [ -d track-record ]; then cp -R track-record "$package_root/"; fi
  # Ship the adapter sources without local dependencies, credentials or runtime state.
  mkdir -p "$package_root/adapters/agent"
  cp adapters/agent/package.json adapters/agent/package-lock.json adapters/agent/README.md adapters/agent/pi-extension.js "$package_root/adapters/agent/"
  cp -R adapters/agent/src adapters/agent/test "$package_root/adapters/agent/"
  mkdir -p "$package_root/adapters/e2b"
  cp adapters/e2b/package.json adapters/e2b/package-lock.json adapters/e2b/bridge.mjs adapters/e2b/protocol.mjs adapters/e2b/remote_tools.py adapters/e2b/remote_mcp.mjs adapters/e2b/runtime-path.mjs adapters/e2b/README.md "$package_root/adapters/e2b/"
  (cd "$BLACKSAND_PACKAGE_DIR" && zip -q -r -9 "$(basename "$archive")" "$(basename "$package_root")")
  rm -rf "$package_root"
  ok "Release archive: $archive"
}

build_target() {
  target="$1"
  case "$target" in
    */*) ;;
    *) die "Invalid target: $target (expected OS/ARCH)" ;;
  esac
  goos="${target%%/*}"
  goarch="${target##*/}"
  case "$goos" in
    linux|darwin|windows) ;;
    *) die "Unsupported OS: $goos (use linux, darwin, windows)" ;;
  esac

  binary_name="blacksand"
  [ "$goos" = "windows" ] && binary_name="blacksand.exe"
  if [ -n "${BLACKSAND_OUTPUT:-}" ] && [ "$BLACKSAND_RELEASE" != "1" ]; then
    output="$BLACKSAND_OUTPUT"
  else
    output="${BLACKSAND_OUTPUT_DIR}/blacksand-${goos}-${goarch}/${binary_name}"
  fi
  mkdir -p "$(dirname "$output")"

  info "Building ${goos}/${goarch}, version ${BLACKSAND_BUILD_VERSION}"
  GOSUMDB="$BLACKSAND_GOSUMDB" \
  CGO_ENABLED=0 \
  GOOS="$goos" \
  GOARCH="$goarch" \
  go build \
    -tags embedui \
    -trimpath \
    -ldflags "-s -w -buildid= -X main.version=${BLACKSAND_BUILD_VERSION}" \
    -o "$output" \
    ./cmd/blacksand

  compress_binary "$output" "$goos"
  if command -v file >/dev/null 2>&1; then file "$output"; fi
  if [ "$BLACKSAND_PACKAGE" = "1" ]; then package_binary "$output" "$goos" "$goarch"; fi
  ok "Build complete: $output"
}

write_checksums() {
  [ "$BLACKSAND_PACKAGE" = "1" ] || return 0
  checksum_file="$BLACKSAND_PACKAGE_DIR/SHA256SUMS"
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$BLACKSAND_PACKAGE_DIR" && for archive in *.zip; do sha256sum "$archive"; done > "$(basename "$checksum_file")")
  elif command -v shasum >/dev/null 2>&1; then
    (cd "$BLACKSAND_PACKAGE_DIR" && for archive in *.zip; do shasum -a 256 "$archive"; done > "$(basename "$checksum_file")")
  else
    warn "sha256sum and shasum unavailable; skipping SHA256SUMS"
    return 0
  fi
  ok "Checksum file: $checksum_file"
}

mkdir -p "$BLACKSAND_OUTPUT_DIR"
if [ "$BLACKSAND_PACKAGE" = "1" ]; then mkdir -p "$BLACKSAND_PACKAGE_DIR"; fi

old_ifs="$IFS"
IFS=','
read -r -a targets <<< "$BLACKSAND_TARGETS"
IFS="$old_ifs"
[ "${#targets[@]}" -gt 0 ] || die "BLACKSAND_TARGETS cannot be empty"
for target in "${targets[@]}"; do
  target="${target//[[:space:]]/}"
  [ -n "$target" ] || continue
  build_target "$target"
done

if [ "$BLACKSAND_PACKAGE" = "1" ]; then
  write_checksums
  info "Release archives generated in: $BLACKSAND_PACKAGE_DIR"
fi
