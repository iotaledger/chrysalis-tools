# Null outputs

Attempts to overflow the balances.
This fails due to protections in iota.go.

```
Usage:
go run overflow/overflow.go

flags:
  -node string
        Can be either domain name or ip of the node (default "127.0.0.1")
  -port int
        Api port (default 14265)
```