#!/usr/bin/env bash
ED25519_PRIVATE_SEED="$(base64 < ed25519_seed.bin | tr -d '\n')" ./scripts/build-all.sh
unset ED25519_PRIVATE_SEED
