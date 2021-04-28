# Genesis Snapshot Network ID Incorporater

Takes a genesis snapshot for Chrysalis Phase 2 and generates a new one where the placeholder transaction IDs get the
specified network ID included.

This tool mainly exists so that a network bootstrapping from such modified genesis snapshot, is no longer subject to
replay attacks, since the output IDs of the "genesis outputs" are different and therefore the signatures are not
applicable in the other network.

The source file is not modified in place, a new file is generated.

Example output:

```
2021/04/28 19:25:31 converting genesis_snapshot_alt.bin to mod_genesis_snapshot_alt.bin by applying as-network to the genesis output tx IDs
2021/04/28 19:25:31 converted, took 34.685016ms
```

Flags:

```
Usage:
  -network-id string
        the name of the network to incorporate into the outputs (default "as-network")
  -source-file string
        the name of the genesis snapshot file to alter (default "genesis_snapshot_alt.bin")
  -target-file string
        the name of the modified genesis snapshot (default "mod_genesis_snapshot_alt.bin")
```