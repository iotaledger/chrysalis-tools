# Snapshot Verify Tool

This tool:

1. queries legacy Hornet nodes for their ledger state
2. computes a Blake2b-256 hash of the sorted [addr,balance] tuples
3. generates a global snapshot file in CSV format to consume for legacy nodes
4. generates a genesis snapshot for Chrysalis Phase 2 nodes containing already burned/migrated funds

Migrated funds are allocated in the genesis snapshot under UTXO IDs with an empty transaction hash and up to index 126.
When the output index goes over 126 it is wrapped to zero, and the transaction ID's last 2 bytes (holds a little endian
encoded uint16) is incremented on each wrap around. The message ID associated with the output is all zero.

Requirements:
- The legacy node must have the `getLedgerState` API command enabled.

#### Usage

Run the tool with `--help` to get a list of configurable CLI params.
It is important to declare the right network ID for the network which is supposed to bootstrap
from the given Chrysalis Phase 2 genesis snapshot file: aka adjust `-genesis-snapshot-file-network-id="<network_id>"` accordingly.