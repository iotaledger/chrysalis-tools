# Migration API

Features:

- Exposes an HTTP API service which can be queried to get information about the migration process.
- Exposes prometheus metrics to be used for graphing the continuous state of the migration process.

## HTTP API service

This service queries legacy and C2 nodes in order to offer a HTTP API to get information of the migration process.
See [openapi.yml](https://editor.swagger.io/?url=https://raw.githubusercontent.com/iotaledger/chrysalis-tools/master/migration-api/openapi.yaml)
for the Open API Specification.

Node requirements:

- The legacy node must allow following HTTP API commands:
    - `getNodeInfo`
    - `getLedgerState`
    - `getLedgerDiffExt`
- The C2 node must allow HTTP API routes:
    - `/api/v1/info`
    - `/api/v1/receipts`
    - `/api/v1/receipts/:migratedAt`
    - `/api/v1/treasury`

## Prometheus Metrics Service

This stateful service keeps track of the amount of included tail transactions on a legacy network, the amount of applied
receipt entries in a C2 network and exposes them as following prometheus counters (names can be altered
in `config.json`):

* `iota_wf_tails_included`
* `iota_receipts_entries_applied`
* `iota_prom_metrics_service_errors` (used for error alerting of this service)

Node requirements:

- The legacy node must allow following HTTP API commands:
    - `getNodeInfo`
    - `getWhiteFlagConfirmation`
- The C2 node must allow HTTP API routes:
    - `/api/v1/info`
    - `/api/v1/milestones/:milestoneIndex`
    - `/api/v1/messages/:messageID/raw`

Configure `promMetricsService.legacyMilestoneStartIndex` and `promMetricsService.c2MilestoneStartIndex` accordingly
before starting the service for the first time. (*The first queries will happen at +1 the configured values.*)

Subsequent restarts of the service will use the state persisted in `prom_metrics_service.state`.

Note that the service also needs to be `promMetricsService.enabled`.

## Docker

Build a docker container with `docker build -t migration-api:dev .` and either alter config.json beforehand or mount it
into the container at `/app/config.json`. Per default, the services listens on `0.0.0.0:8484`. Make sure to also
mount `prom_metrics_service.state` onto the host system in order to persist the `Prometheus Metrics Service` state.