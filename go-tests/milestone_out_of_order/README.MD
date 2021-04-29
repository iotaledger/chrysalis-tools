#Milestones out of order

This attack generates milestones with indexes out of order that confirm conflicting value transactions. The point is to imitate a condition where nodes receive milestones out of order.

```
Usage:
go run milestone_out_of_order/milestone_out_of_order.go

flags:
  -mqtt int
        Mqtt port (default 1883)
  -node string
        Can be either domain name or ip of the node (default "127.0.0.1")
  -port int
        Api port (default 14265)
```