# Null outputs

Simply tests what happens when we transfer to a null address.
Balance actually transfers but no errors occur

```
Usage:
go run null_outputs/transfer_to_null.go

flags:
  -node string
        Can be either domain name or ip of the node (default "127.0.0.1")
  -port int
        Api port (default 14265)
```