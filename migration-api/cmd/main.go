package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/iotaledger/chrysalis-tools/migration-api"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	cfg, err := migration_api.ReadConfig()
	must(err)

	sigs := make(chan os.Signal, 1)
	done := make(chan struct{}, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		if err := migration_api.Shutdown(); err != nil {
			log.Panic(err)
		}
		done <- struct{}{}
	}()

	if err := migration_api.Start(cfg); err != nil {
		log.Panic(err)
	}

	<-done
}
