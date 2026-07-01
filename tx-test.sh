#!/bin/bash
set -euo pipefail

# Default: mainnet node
NODE=${MONERO_NODE:-"https://xmr1.doggett.tech:18089"}
NETWORK=${1:-mainnet}

echo "==> Fetching latest block from $NODE (network: $NETWORK)..."

# Get latest block height
HEIGHT=$(curl -s -X POST "$NODE/json_rpc" \
    -d '{"jsonrpc":"2.0","id":"0","method":"get_block_count"}' \
    -H 'Content-Type: application/json' | jq -r '.result.count')
echo "Latest block height: $HEIGHT"

TX_HASH=""
SEARCH_HEIGHT=$HEIGHT

# Search backwards until we find a transaction
while [ "$SEARCH_HEIGHT" -ge 0 ]; do
    echo "Checking block height $SEARCH_HEIGHT..."
    BLOCK_JSON=$(curl -s -X POST "$NODE/json_rpc" \
        -d "{\"jsonrpc\":\"2.0\",\"id\":\"0\",\"method\":\"get_block\",\"params\":{\"height\":$SEARCH_HEIGHT}}" \
        -H 'Content-Type: application/json')
    
    # Extract first non-coinbase transaction hash (skip miner_tx)
    TX_HASH=$(echo "$BLOCK_JSON" | jq -r '.result.tx_hashes[0]')
    if [ "$TX_HASH" != "null" ] && [ -n "$TX_HASH" ]; then
        echo "Found transaction in block $SEARCH_HEIGHT: $TX_HASH"
        break
    fi
    
    # Move to previous block
    SEARCH_HEIGHT=$((SEARCH_HEIGHT - 1))
    
    # Safety: prevent infinite loop if we go back too far
    if [ "$SEARCH_HEIGHT" -lt $((HEIGHT - 1000)) ]; then
        echo "No transaction found in the last 1000 blocks."
        exit 1
    fi
done

if [ -z "$TX_HASH" ]; then
    echo "No transaction found (only coinbase blocks)."
    exit 1
fi

# Fetch transaction blob as hex
TX_BLOB=$(curl -s -X POST "$NODE/get_transactions" \
    -d "{\"txs_hashes\":[\"$TX_HASH\"],\"decode_as_json\":false}" \
    -H 'Content-Type: application/json' | jq -r '.txs[0].as_hex')

if [ -z "$TX_BLOB" ] || [ "$TX_BLOB" == "null" ]; then
    echo "Failed to get transaction blob."
    exit 1
fi

# Prepare testdata directory
mkdir -p testdata
TEST_FILE="testdata/transaction.bin"

# Backup existing test file if present
if [ -f "$TEST_FILE" ]; then
    BACKUP_FILE="${TEST_FILE}.bak"
    mv "$TEST_FILE" "$BACKUP_FILE"
    echo "Backed up existing testdata to $BACKUP_FILE"
fi

# Write new blob
echo "$TX_BLOB" | xxd -r -p > "$TEST_FILE"
echo "Written new test blob to $TEST_FILE"

# Run the parse test
echo "==> Running go-monblob parse test..."
go test -v -run=^TestParseKnownTransaction$

echo "==> Test completed successfully."