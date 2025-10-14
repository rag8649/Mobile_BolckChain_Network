#!/bin/bash
set -e

HOME_DIR=./private/.simapp
CHAIN_ID=learning-chain-1
KEY_NAME=alice
KEYRING=test
DENOM=stake

echo "=== 1️⃣ Init blockchain ==="
./build/simd init demo \
  --home $HOME_DIR \
  --chain-id $CHAIN_ID

echo "=== 2️⃣ Add key ($KEY_NAME) ==="
./build/simd keys add $KEY_NAME \
  --home $HOME_DIR \
  --keyring-backend $KEYRING

echo "=== 3️⃣ Add genesis account ==="
./build/simd add-genesis-account $KEY_NAME 100000000${DENOM} \
  --home $HOME_DIR \
  --keyring-backend $KEYRING

echo "=== 4️⃣ Generate staking transaction ==="
./build/simd gentx $KEY_NAME 70000000${DENOM} \
  --home $HOME_DIR \
  --keyring-backend $KEYRING \
  --chain-id $CHAIN_ID

echo "=== 5️⃣ Collect gentxs ==="
./build/simd collect-gentxs \
  --home $HOME_DIR

echo "=== 6️⃣ Start node ==="
./build/simd start \
  --home $HOME_DIR
