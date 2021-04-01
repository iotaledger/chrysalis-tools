# End-to-End Migration (testing) Tools

### Generate a global snapshot file with X allocated addresses for legacy nodes

Example:

```
./e2e -mode="generateGSAddresses" -gs-addrs-count=100000 -gs-seed-map-file="seedmap.csv" -gs-snapshot-file-file="snapshot.csv"
```

This generates a `snapshot.csv` containing 100000 addresses with a proportional amount of tokens in relation to the
total supply (last address may contain the remainder instead if supply can't be split up evenly). The
corresponding `seedmap.csv` includes the mapping of seeds to their first address (security level 2) and the funds
residing on them.

### Generate and broadcast migration bundles via a seedmap.csv

Example:

```
./e2e -mode="migrate" -nodeAPIURI="https://<legacy-node-api-uri>" -migration-source-file="seedmap.csv" -migration-info-file="migrated.csv" -migration-from=0 -migration-to=1000 -migration-batch-size="40" 
```

This would start generating migration bundles from entries 0 to 1000 from `seedmap.csv` and broadcasting them in batches
of 40 to the specified node. The specified `migrated.csv` contains the tail transaction hash, private key of the Ed25519
addr (hex), target Ed25519 addr (hex) and the migrated funds.