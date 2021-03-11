module github.com/iotaledger/chrysalis-tools/migration-api

go 1.16

replace github.com/iotaledger/chrysalis-tools/common => ../common

require (
	github.com/iotaledger/chrysalis-tools/common v0.0.0-20210310095909-b38c905df767
	github.com/iotaledger/iota.go v1.0.0-beta.15.0.20210212090247-51c40bcebea7
	github.com/iotaledger/iota.go/v2 v2.0.0-20210309092402-a4a03ab62bd2
	github.com/labstack/echo/v4 v4.2.0
	github.com/prometheus/client_golang v1.9.0 // indirect
	github.com/spf13/viper v1.7.1
)
