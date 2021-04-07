# End-to-End Migration SeedMap Tool

Generates a global snapshot file with X allocated addresses for legacy nodes:

Example:

```
./seedmap -addrs-count=100000 -seed-map-file="seedmap.csv" -snapshot-file-file="snapshot.csv"
```

This generates a `snapshot.csv` containing 100000 addresses with a proportional amount of tokens in relation to the
total supply (last address may contain the remainder instead if supply can't be split up evenly). The
corresponding `seedmap.csv` includes the mapping of seeds to their first address (security level 2) and the funds
residing on them.