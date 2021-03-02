# Migration API

This service offers an API to query for migration related information. Mainly used for the migration transparency site.
It connects to a legacy and a C2 Hornet node to fetch data and prepares it for consumption.

See [openapi.yml](https://editor.swagger.io/?https://raw.githubusercontent.com/iotaledger/chrysalis-tools/master/migration-api/openapi.yaml)
for the Open API Specification.

## Setup

- The legacy node must allow HTTP API calls to `getLedgerState` and `getLedgerDiffExt`.
- The C2 node must allow HTTP API calls to `/receipts`/`/receipts/:migratedAt`/`/treasury`

Build a docker container with `docker build -t migration-api:dev .` and either alter config.json beforehand or mount it
into the container at `/app/config.json`. Per default, the service listens on `0.0.0.0:8484`
