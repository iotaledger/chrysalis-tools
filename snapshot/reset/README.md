# Snapshot Reset Tool

This tool queries a Chrysalis Phase 2 node for its UTXOs and treasury and then generates a full snapshot containing
those. This tool is mainly used to reset a network but keeping its current ledger state.

The queried node must support following routes for the tool to function:

* `/api/plugins/debug/outputs/unspent`
* `/api/v1/outputs/:outputID`
* `/api/v1/treasury`
