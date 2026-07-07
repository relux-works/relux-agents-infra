#!/usr/bin/env zsh

set -euo pipefail

SOURCE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BINARY_NAME="agents-infra"
BUILD_OUTPUT="$SOURCE_DIR/.temp/bin/$BINARY_NAME"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"
WITH_PDF_TOOLS=0
CONFIG_DIR="${AGENTS_INFRA_CONFIG_DIR:-}"
INSTALL_STATE_PATH=""
BUILD_VERSION="dev"
BUILD_COMMIT="unknown"
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
BUILD_LDFLAGS=""
LLDB_MCP_WRAPPER_MARKER="agents-infra managed lldb-mcp wrapper"

green() { print -P "%F{green}$1%f"; }
yellow() { print -P "%F{yellow}$1%f"; }
red() { print -P "%F{red}$1%f"; }

json_escape() {
  print -rn -- "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

usage() {
  cat <<EOF
Usage: scripts/setup.sh [options]

Options:
  --bin-dir PATH       Install the agents-infra binary into PATH (default: $HOME/.local/bin)
  --with-pdf-tools     Install optional PDF toolchain (pandoc, weasyprint, poppler)
  --help, -h           Show this help

Environment:
  AGENTS_INFRA_SKIP_LLDB_MCP=1  Skip macOS Homebrew llvm/lldb-mcp setup
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --bin-dir)
      BIN_DIR="$2"
      shift 2
      ;;
    --with-pdf-tools)
      WITH_PDF_TOOLS=1
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      red "Unknown option: $1"
      usage
      exit 1
      ;;
  esac
done

resolve_config_dir() {
  if [[ -n "$CONFIG_DIR" ]]; then
    return
  fi
  if [[ "$(uname -s)" == "Darwin" ]]; then
    CONFIG_DIR="$HOME/Library/Application Support/agents-infra"
    return
  fi
  CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/agents-infra"
}

install_go() {
  if command -v go >/dev/null 2>&1; then
    green "Go already installed: $(go version)"
    return
  fi

  if [[ "$(uname -s)" == "Darwin" ]] && command -v brew >/dev/null 2>&1; then
    yellow "Go not found. Installing via Homebrew..."
    brew install go
    green "Go installed: $(go version)"
    return
  fi

  red "Go is missing and automatic install is unavailable on this platform."
  red "Install Go manually and rerun setup."
  exit 1
}

install_lldb_mcp() {
  if [[ "${AGENTS_INFRA_SKIP_LLDB_MCP:-0}" == "1" ]]; then
    yellow "Skipping lldb-mcp setup because AGENTS_INFRA_SKIP_LLDB_MCP=1"
    return
  fi

  if [[ "$(uname -s)" != "Darwin" ]]; then
    return
  fi

  if ! command -v brew >/dev/null 2>&1; then
    yellow "Homebrew not found; skipping lldb-mcp setup."
    yellow "Install Homebrew or provide lldb-mcp on PATH for LLDB MCP support."
    return
  fi

  local brew_prefix wrapper existing
  brew_prefix="$(brew --prefix)"
  wrapper="$brew_prefix/bin/lldb-mcp"
  existing="$(command -v lldb-mcp 2>/dev/null || true)"

  if [[ -n "$existing" && "$existing" != "$wrapper" ]]; then
    green "lldb-mcp already installed: $existing"
    return
  fi

  yellow "Ensuring Homebrew llvm and lldb-mcp wrapper for LLDB MCP support..."
  brew install llvm

  local llvm_prefix helper
  llvm_prefix="$(brew --prefix llvm)"
  helper="$llvm_prefix/bin/lldb-mcp"
  if [[ ! -x "$helper" ]]; then
    red "Homebrew llvm did not provide expected helper: $helper"
    exit 1
  fi

  if [[ ! -x "$llvm_prefix/bin/lldb" ]]; then
    red "Homebrew llvm did not provide expected lldb: $llvm_prefix/bin/lldb"
    exit 1
  fi

  mkdir -p "$(dirname "$wrapper")"
  if [[ -e "$wrapper" && ! -L "$wrapper" ]]; then
    local backup="$wrapper.agents-infra.bak"
    if [[ ! -e "$backup" ]]; then
      cp "$wrapper" "$backup"
      yellow "Backed up existing lldb-mcp at $wrapper to $backup"
    else
      yellow "Existing lldb-mcp backup already present: $backup"
    fi
  fi
  rm -f "$wrapper"
cat > "$wrapper" <<EOF
#!/bin/sh
# $LLDB_MCP_WRAPPER_MARKER.
set -eu

state_dir="\${HOME:-}/.lldb"
if [ -n "\${HOME:-}" ] && [ -d "\$state_dir" ]; then
  for state_file in "\$state_dir"/lldb-mcp-*.json; do
    [ -e "\$state_file" ] || continue
    state_name=\${state_file##*/}
    state_pid=\${state_name#lldb-mcp-}
    state_pid=\${state_pid%.json}
    case "\$state_pid" in
      ''|*[!0-9]*) continue ;;
    esac
    if ! kill -0 "\$state_pid" 2>/dev/null; then
      rm -f "\$state_file"
    fi
  done
fi

exec "$helper" "\$@"
EOF
  chmod +x "$wrapper"
  green "Installed lldb-mcp wrapper: $wrapper"
}

compute_ldflags() {
  if git -C "$SOURCE_DIR" rev-parse --git-dir >/dev/null 2>&1; then
    BUILD_VERSION="$(git -C "$SOURCE_DIR" describe --tags --always 2>/dev/null || echo "dev")"
    BUILD_COMMIT="$(git -C "$SOURCE_DIR" rev-parse --short HEAD 2>/dev/null || echo "unknown")"
  fi
  BUILD_LDFLAGS="-X main.Version=$BUILD_VERSION -X main.Commit=$BUILD_COMMIT -X main.BuildDate=$BUILD_DATE"
}

build_cli() {
  green "Building $BINARY_NAME ..."
  mkdir -p "$(dirname "$BUILD_OUTPUT")"
  go -C "$SOURCE_DIR/tools/agents-infra" build -trimpath -ldflags "$BUILD_LDFLAGS" -o "$BUILD_OUTPUT" .
  green "Built: $BUILD_OUTPUT"
}

install_binary() {
  local dest="$BIN_DIR/$BINARY_NAME"
  local tmp="$dest.tmp.$$"
  mkdir -p "$BIN_DIR"
  cp "$BUILD_OUTPUT" "$tmp"
  chmod +x "$tmp"
  mv -f "$tmp" "$dest"
  green "Installed binary: $dest"
}

write_install_state() {
  resolve_config_dir
  INSTALL_STATE_PATH="$CONFIG_DIR/install.json"
  mkdir -p "$CONFIG_DIR"

  local escaped_repo escaped_bin escaped_platform escaped_arch escaped_version escaped_commit escaped_build_date
  escaped_repo="$(json_escape "$SOURCE_DIR")"
  escaped_bin="$(json_escape "$BIN_DIR")"
  escaped_platform="$(json_escape "$(uname -s | tr '[:upper:]' '[:lower:]')")"
  escaped_arch="$(json_escape "$(uname -m)")"
  escaped_version="$(json_escape "$BUILD_VERSION")"
  escaped_commit="$(json_escape "$BUILD_COMMIT")"
  escaped_build_date="$(json_escape "$BUILD_DATE")"
  cat > "$INSTALL_STATE_PATH" <<EOF
{
  "repoPath": "$escaped_repo",
  "binDir": "$escaped_bin",
  "platform": "$escaped_platform",
  "arch": "$escaped_arch",
  "version": "$escaped_version",
  "commit": "$escaped_commit",
  "buildDate": "$escaped_build_date"
}
EOF
  green "Install state: $INSTALL_STATE_PATH"
}

ensure_user_path() {
  if command -v "$BINARY_NAME" >/dev/null 2>&1; then
    return
  fi
  yellow "$BIN_DIR is not in PATH yet."
  yellow "Add to your shell config: export PATH=\"$BIN_DIR:\$PATH\""
}

verify_install() {
  local dest="$BIN_DIR/$BINARY_NAME"
  [[ -x "$dest" ]] || { red "Missing installed binary: $dest"; exit 1; }
  "$dest" version >/dev/null
  "$dest" setup global --source-dir "$SOURCE_DIR" >/dev/null
  "$dest" doctor global >/dev/null
  green "Verified binary and global setup"
}

print ""
green "=== alexis-agents-infra setup ==="
print ""
install_go
install_lldb_mcp
compute_ldflags
build_cli
install_binary
write_install_state
if [[ "$WITH_PDF_TOOLS" == "1" ]]; then
  green "Installing optional PDF toolchain"
  "$SOURCE_DIR/.scripts/setup-pdf-tools.sh"
fi
ensure_user_path
verify_install
print ""
green "=== Done ==="
