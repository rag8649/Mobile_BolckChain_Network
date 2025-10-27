#!/bin/bash
set -e

echo "=== 1️⃣ Init blockchain ==="

make build

./build/simd init demo \
    --home ./private/.simapp \
    --chain-id learning-chain-1

./build/simd keys list \
    --home ./private/.simapp \
    --keyring-backend test

echo "=== 2️⃣ Add key ($KEY_NAME) ==="
./build/simd keys add alice \
    --home ./private/.simapp \
    --keyring-backend test


echo "=== 3️⃣ Add genesis account ==="
./build/simd add-genesis-account alice 100000000000stake \
    --home ./private/.simapp \
    --keyring-backend test



echo "=== 4️⃣ Generate staking transaction ==="
./build/simd gentx alice 70000000stake \
    --home ./private/.simapp \
    --keyring-backend test \
    --chain-id learning-chain-1

echo "=== 5️⃣ Collect gentxs ==="
 ./build/simd collect-gentxs \
    --home ./private/.simapp

