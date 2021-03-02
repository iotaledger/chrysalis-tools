package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/iotaledger/chrysalis-tools/migration-api/api"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	cfg, err := api.ReadConfig()
	must(err)

	log.Printf("booting up server with following config: %s", cfg.JSONString())

	sigs := make(chan os.Signal, 1)
	done := make(chan struct{}, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		log.Println("got sigint/sigterm. shutting down...")
		if err := api.Shutdown(); err != nil {
			log.Panic(err)
		}
		done <- struct{}{}
	}()

	if err := api.Start(cfg); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Panic(err)
	}

	log.Println("service shutdown successfully")
	<-done
}
