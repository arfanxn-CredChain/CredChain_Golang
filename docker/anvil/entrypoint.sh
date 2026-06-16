#!/bin/sh
set -e

STATE_FILE="/state/state.json"

if [ -f "${STATE_FILE}" ]; then
    echo "[anvil] Loading existing state from ${STATE_FILE}"
    exec anvil \
        --host 0.0.0.0 \
        --port 8545 \
        --mnemonic "test test test test test test test test test test test junk" \
        --dump-state "${STATE_FILE}" \
        --load-state "${STATE_FILE}"
else
    echo "[anvil] No state file found — starting fresh chain"
    exec anvil \
        --host 0.0.0.0 \
        --port 8545 \
        --mnemonic "test test test test test test test test test test test junk" \
        --dump-state "${STATE_FILE}"
fi
