#!/usr/bin/env python3
"""Deploy Solidity contracts and write their addresses into a Go env file.

Target env file defaults to .env.docker (full-Docker/prod). For the local
hybrid flow, run with ENV_FILE=.env so the host backend picks up the addresses:

    ENV_FILE=.env python3 scripts/setup-contracts.py
"""

import json
import os
import subprocess
import sys
from pathlib import Path

# scripts/ -> CredChain_Golang/ -> workspace root
ROOT = Path(__file__).resolve().parent.parent.parent
SOLIDITY_DIR = ROOT / "CredChain_Solidity"
GOLANG_DIR = ROOT / "CredChain_Golang"
ENV_FILE = GOLANG_DIR / os.environ.get("ENV_FILE", ".env.docker")
DEPLOYMENTS_DIR = SOLIDITY_DIR / "deployments"

NETWORK = os.environ.get("NETWORK", "localhost")
DEPLOYMENT_FILE = DEPLOYMENTS_DIR / f"{NETWORK}.json"


def run(cmd: list[str], cwd: Path) -> None:
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
    deploy()
    deployment = read_deployment()
    update_env(deployment)
    print("[setup-contracts] Done.")


if __name__ == "__main__":
    main()
