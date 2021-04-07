# End-to-End Migration (testing) Tools

Generates and broadcast migration bundles via a seedmap.csv file.

Example:

```
./migrate -nodeAPIURI="https://<legacy-node-api-uri>" -source-file="seedmap.csv" -info-file="migrated.csv" -from=0 -to=1000 -batch-size="40" 
```

This would start generating migration bundles from entries 0 to 1000 from `seedmap.csv` and broadcasting them in batches
of 40 to the specified node. The specified `migrated.csv` contains the tail transaction hash, private key of the Ed25519
addr (hex), target Ed25519 addr (hex) and the migrated funds.