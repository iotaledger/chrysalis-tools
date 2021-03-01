module github.com/iotaledger/chrysalis-tools/migration-api

go 1.16

replace github.com/iotaledger/hive.go => github.com/muxxer/hive.go v0.0.0-20210222004711-d924b9529a49

replace github.com/iotaledger/chrysalis-tools => ../

require (
	github.com/iotaledger/chrysalis-tools v0.0.0-00010101000000-000000000000
	github.com/iotaledger/iota.go v1.0.0-beta.15.0.20210212090247-51c40bcebea7
	github.com/iotaledger/iota.go/v2 v2.0.0-20210301150403-555f6dae0fe0
	github.com/labstack/echo/v4 v4.2.0
	github.com/spf13/viper v1.7.1
)
