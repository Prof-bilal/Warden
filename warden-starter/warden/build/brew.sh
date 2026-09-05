#!/usr/bin/env bash
# Generates a Homebrew formula for warden and prints it to stdout.
# Usage: VERSION=1.2.3 ./build/brew.sh [dist_dir] > warden.rb
#
# If no dist_dir is given, falls back to git-referenced download URLs with
# per-OS sha256 placeholders — the CI release job fills those in.
# When run from a checkout with built dist/, it computes real hashes.

set -euo pipefail

VERSION="${VERSION:-dev}"
# Strip leading 'v' if present — Homebrew prefers bare versions like "1.2.3".
BARE_VERSION="${VERSION#v}"
DIST_DIR="${2:-$(dirname "$0")/../dist}"

if [ "$BARE_VERSION" = "dev" ]; then
  BARE_VERSION="$(git -C "$(dirname "$0")/.." describe --tags --always --dirty 2>/dev/null || echo dev)"
fi

# Try to compute real sha256s from locally-built dist/ binaries.
if [ -f "$DIST_DIR/warden-linux-amd64" ] && [ -f "$DIST_DIR/warden-darwin-arm64" ]; then
  LINUX_SHA=$(sha256sum "$DIST_DIR/warden-linux-amd64" | awk '{print $1}')
  DARWIN_SHA=$(shasum -a 256 "$DIST_DIR/warden-darwin-arm64" | awk '{print $1}')

  cat <<EOF
class Warden < Formula
  desc "Sandbox runtime for MCP servers"
  homepage "https://github.com/warden-sandbox/warden"
  version "$BARE_VERSION"

  on_macos do
    url "https://github.com/warden-sandbox/warden/releases/download/v${BARE_VERSION}/warden-darwin-arm64"
    sha256 "$DARWIN_SHA"
  end

  on_linux do
    url "https://github.com/warden-sandbox/warden/releases/download/v${BARE_VERSION}/warden-linux-amd64"
    sha256 "$LINUX_SHA"
  end

  license "MIT"

  def install
    if OS.mac?
      bin.install "warden-darwin-arm64" => "warden"
    else
      bin.install "warden-linux-amd64" => "warden"
    end
  end

  test do
    assert_match(/warden - a sandbox runtime for MCP servers/, shell_output(bin/"warden" 2>&1, status: 1))
  end
end
EOF
else
  # Fallback: formula with placeholder sha256s (filled by CI before commit).
  cat <<EOF
class Warden < Formula
  desc "Sandbox runtime for MCP servers"
  homepage "https://github.com/warden-sandbox/warden"
  version "$BARE_VERSION"

  on_macos do
    url "https://github.com/warden-sandbox/warden/releases/download/v${BARE_VERSION}/warden-darwin-arm64"
    sha256 "TODO: fill with actual sha256"
  end

  on_linux do
    url "https://github.com/warden-sandbox/warden/releases/download/v${BARE_VERSION}/warden-linux-amd64"
    sha256 "TODO: fill with actual sha256"
  end

  license "MIT"

  def install
    if OS.mac?
      bin.install "warden-darwin-arm64" => "warden"
    else
      bin.install "warden-linux-amd64" => "warden"
    end
  end

  test do
    assert_match(/warden - a sandbox runtime for MCP servers/, shell_output(bin/"warden" 2>&1, status: 1))
  end
end
EOF
fi
