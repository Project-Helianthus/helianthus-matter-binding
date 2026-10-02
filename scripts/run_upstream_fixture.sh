#!/usr/bin/env bash
# Run only the SDK's public sample in a self-cleaning temporary directory.
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 SDK_CHECKOUT" >&2
  exit 2
fi

sdk_checkout=$(cd "$1" && pwd)
repo_root=$(cd "$(dirname "$0")/.." && pwd)
app="$sdk_checkout/out/linux-x64-all-devices-clang/all-devices-app"
tool="$sdk_checkout/out/linux-x64-chip-tool-clang/chip-tool"
work=$(mktemp -d)
external_tmp=$(mktemp -d)
app_pid=""

cleanup() {
  if [[ -n "$app_pid" ]] && kill -0 "$app_pid" 2>/dev/null; then
    kill "$app_pid" 2>/dev/null || true
    wait "$app_pid" 2>/dev/null || true
  fi
  rm -rf "$work" "$external_tmp"
}
trap cleanup EXIT INT TERM

[[ -x "$app" ]] || { echo "missing all-devices-app" >&2; exit 1; }
[[ -x "$tool" ]] || { echo "missing chip-tool" >&2; exit 1; }
mkdir -p "$work/home"
printf 'preserve\n' >"$external_tmp/chip_tool_config.ini"

"$app" --device electrical-sensor:1 --KVS "$work/sample.kvs" --discriminator 1234 >"$work/app.log" 2>&1 &
app_pid=$!
for _ in $(seq 1 30); do
  grep -q "Server initialization complete" "$work/app.log" && break
  sleep 1
done
grep -q "Server initialization complete" "$work/app.log"

# The SDK sample documents the public test passcode and uses only this temporary
# controller home. These checks concern the sample's composition, never a
# Helianthus production profile.
TMPDIR="$work" HOME="$work/home" timeout 120 "$tool" pairing onnetwork 1 20202021
TMPDIR="$work" HOME="$work/home" timeout 60 "$tool" descriptor read device-type-list 1 1 | tee "$work/device-types.log"
TMPDIR="$work" HOME="$work/home" timeout 60 "$tool" descriptor read server-list 1 1 | tee "$work/server-list.log"
grep -Eiq '(0x0*510|\b1296\b)' "$work/device-types.log"
grep -Eiq '(0x0*1[dD]|\b29\b)' "$work/server-list.log"
grep -Eiq '(0x0*90|\b144\b)' "$work/server-list.log"
grep -Eiq '(0x0*9[cC]|\b156\b)' "$work/server-list.log"
[[ -s "$work/chip_tool_config.ini" ]]
[[ -s "$work/chip_tool_config.alpha.ini" ]]
[[ -f "$work/chip_tool_kvs" ]]
grep -qx 'preserve' "$external_tmp/chip_tool_config.ini"
kill "$app_pid"
wait "$app_pid" || true
app_pid=""

# The upstream EPM test starts a fresh sample, commissions it through its Python
# controller, triggers the SDK's fake load, and checks numeric ActiveCurrent.
cd "$sdk_checkout"
"$repo_root/scripts/run_owned_group.sh" 300 ./scripts/build_python.sh -i "$work/venv" --enable_ipv4 true
source "$work/venv/bin/activate"
"$repo_root/scripts/run_owned_group.sh" 300 ./scripts/tests/run_python_test.py \
  --app "$app" \
  --app-args "--device electrical-sensor:1 --discriminator 1234 --KVS $work/fixture.kvs --enable-key 000102030405060708090a0b0c0d0e0f" \
  --script src/python_testing/TC_EPM_2_2.py \
  --script-args "--storage-path $work/admin_storage.json --commissioning-method on-network --discriminator 1234 --passcode 20202021 --hex-arg enableKey:000102030405060708090a0b0c0d0e0f --endpoint 1 --PICS src/app/tests/suites/certification/ci-pics-values"
