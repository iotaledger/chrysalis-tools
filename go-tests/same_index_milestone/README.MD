# Same index milestone generation

Creates two milestones with the same index and send them to the node.
Attempts to mimick situations that can happen in private nets or testnets.

```
Usage:
go run same_index_milestone/same_index.go

flags:
  -node string
        Can be either domain name or ip of the node (default "127.0.0.1")
  -port int
        Api port (default 14265)
```