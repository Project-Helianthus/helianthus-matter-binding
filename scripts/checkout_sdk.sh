#!/usr/bin/env bash
# Fetch the reviewed SDK object without relying on a branch tip or local cache.
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 SDK_CHECKOUT" >&2
  exit 2
fi

repo_root=$(cd "$(dirname "$0")/.." && pwd)
sdk_checkout=$1
sdk_commit=29b4768a513cf566011ab8cd60df1bc495204953
sdk_remote=https://github.com/AryaHassanli/connectedhomeip.git

if [[ -e "$sdk_checkout" ]]; then
  echo "SDK checkout destination already exists: $sdk_checkout" >&2
  exit 2
fi

git init "$sdk_checkout"
git -C "$sdk_checkout" remote add origin "$sdk_remote"
git -C "$sdk_checkout" -c protocol.version=2 fetch --no-tags --depth=1 origin "$sdk_commit"
git -C "$sdk_checkout" checkout --detach FETCH_HEAD
git -C "$sdk_checkout" submodule sync --recursive
git -C "$sdk_checkout" -c protocol.file.allow=never submodule update --init --recursive --depth 1

python3 "$repo_root/tools/validate_lock.py" --sdk-checkout "$sdk_checkout"
