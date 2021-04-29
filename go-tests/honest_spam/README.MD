#Honest Spam

Sends in data messages using the default api. It keeps track of the confirmation rate via MQTT and prints it. This tool should be used in parallel to other attacks running on the tangle.

```
Usage:
go run honest_spam/honest.go

flags:
  -mqtt int
        Mqtt port (default 1883)
  -node string
        Can be either domain name or ip of the node (default "127.0.0.1")
  -port int
        Api port (default 14265)
```