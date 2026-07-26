#!/usr/bin/env python3
"""Deploy Solidity contracts and write their addresses into a Go env file.

Idempotent: if the contracts named in the target env file already have bytecode
on-chain (persisted Anvil state), deployment is skipped. Only a fresh chain
triggers an actual deploy.

Target env file defaults to .env.docker (full-Docker/prod). For the local
hybrid flow, run with ENV_FILE=.env:

    ENV_FILE=.env python3 scripts/setup-contracts.py

This script runs on the HOST, so it reaches the Anvil container over its
published port (127.0.0.1:8545), not the in-network name `anvil`. Override with
RPC_URL if needed.
"""

import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

# scripts/ -> CredChain_Golang/ -> workspace root
ROOT = Path(__file__).resolve().parent.parent.parent
SOLIDITY_DIR = ROOT / "CredChain_Solidity"
GOLANG_DIR = ROOT / "CredChain_Golang"
ENV_FILE = GOLANG_DIR / os.environ.get("ENV_FILE", ".env.docker")
DEPLOYMENTS_DIR = SOLIDITY_DIR / "deployments"

NETWORK = os.environ.get("NETWORK", "localhost")
DEPLOYMENT_FILE = DEPLOYMENTS_DIR / f"{NETWORK}.json"

# Host RPC: this script runs on the host, so it reaches Anvil via its published
# port, not the container-network name `anvil`.
RPC_URL = os.environ.get("RPC_URL", "http://127.0.0.1:8545")


def rpc_call(method: str, params: list, rpc_url: str = RPC_URL) -> dict:
    payload = json.dumps(
        {"jsonrpc": "2.0", "id": 1, "method": method, "params": params}
    ).encode()
    req = urllib.request.Request(
        rpc_url, data=payload, headers={"Content-Type": "application/json"}
    )
    with urllib.request.urlopen(req, timeout=5) as resp:
        return json.loads(resp.read())


def wait_for_anvil(rpc_url: str = RPC_URL, timeout: int = 60) -> None:
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            rpc_call("eth_blockNumber", [], rpc_url)
            print(f"[setup-contracts] Anvil reachable at {rpc_url}")
            return
        except (urllib.error.URLError, OSError):
            time.sleep(1)
    print(f"[setup-contracts] ERROR: Anvil not reachable at {rpc_url} after {timeout}s")
    sys.exit(1)


def get_code(address: str) -> str:
    return rpc_call("eth_getCode", [address, "latest"]).get("result", "0x")


def has_code(address: str, get_code_fn) -> bool:
    if not address:
        return False
    code = get_code_fn(address)
    return bool(code) and code not in ("0x", "0x0")


def read_env_addresses(env_path: Path) -> dict:
    addrs = {}
    if not env_path.exists():
        return addrs
    for line in env_path.read_text().splitlines():
        if line.startswith("AUTHORITY_CONTRACT="):
            addrs["authority"] = line.split("=", 1)[1].strip()
        elif line.startswith("REGISTRY_CONTRACT="):
            addrs["registry"] = line.split("=", 1)[1].strip()
    return addrs


def contracts_exist(addresses: dict, get_code_fn=get_code) -> bool:
    if not addresses.get("authority") or not addresses.get("registry"):
        return False
    return has_code(addresses["authority"], get_code_fn) and has_code(
        addresses["registry"], get_code_fn
    )


def run(cmd: list, cwd: Path) -> None:
    print(f"[setup-contracts] cd {cwd} && {' '.join(cmd)}")
    result = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True)
    if result.returncode != 0:
        print(f"[setup-contracts] ERROR:\n{result.stderr}")
        sys.exit(1)
    print(result.stdout)


def deploy() -> None:
    run(["npm", "ci"], SOLIDITY_DIR)
    run(["npx", "hardhat", "run", "scripts/deploy.ts", "--network", NETWORK], SOLIDITY_DIR)


def read_deployment() -> dict:
    if not DEPLOYMENT_FILE.exists():
        print(f"[setup-contracts] ERROR: deployment file not found: {DEPLOYMENT_FILE}")
        sys.exit(1)
    with open(DEPLOYMENT_FILE) as f:
        return json.load(f)


def update_env(deployment: dict) -> None:
    authority = deployment["credentialAuthority"]
    registry = deployment["credentialRegistry"]

    if not ENV_FILE.exists():
        print(f"[setup-contracts] ERROR: env file not found: {ENV_FILE}")
        sys.exit(1)

    lines = ENV_FILE.read_text().splitlines()
    updated = []
    for line in lines:
        if line.startswith("AUTHORITY_CONTRACT="):
            updated.append(f"AUTHORITY_CONTRACT={authority}")
        elif line.startswith("REGISTRY_CONTRACT="):
            updated.append(f"REGISTRY_CONTRACT={registry}")
        else:
            updated.append(line)

    ENV_FILE.write_text("\n".join(updated) + "\n")
    print(f"[setup-contracts] Updated {ENV_FILE}")
    print(f"  AUTHORITY_CONTRACT={authority}")
    print(f"  REGISTRY_CONTRACT={registry}")


def main() -> None:
    wait_for_anvil()
    existing = read_env_addresses(ENV_FILE)
    if contracts_exist(existing):
        print(
            f"[setup-contracts] Contracts already deployed "
            f"(authority={existing.get('authority')}, registry={existing.get('registry')}); skipping."
        )
        print("[setup-contracts] Done.")
        return
    deploy()
    deployment = read_deployment()
    update_env(deployment)
    print("[setup-contracts] Done.")


if __name__ == "__main__":
    main()
