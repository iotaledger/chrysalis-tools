# Migration API

This service offers an API to query for migration related information. Mainly used for the migration transparency site.
It connects to a legacy and a C2 Hornet node to fetch data and prepares it for consumption.

See [openapi.yml](https://editor.swagger.io/?url=https://github.com/iotaledger/chrysalis-tools/raw/master/migration-api/openapi.yaml)
for the Open API Specification.

## Setup

- The legacy node must allow HTTP API calls to `getLedgerState` and `getLedgerDiffExt`.
- The C2 node must allow HTTP API calls to `/receipts`/`/receipts/:migratedAt`/`/treasury`

