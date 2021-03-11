package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/iotaledger/chrysalis-tools/migration-api/migration"
	"github.com/labstack/echo/v4"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	cfg, err := migration.ReadConfig()
	must(err)

	log.Printf("booting up server with following config: %s", cfg.JSONString())

	e := echo.New()
	e.Debug = true
	e.HideBanner = true

	var services []service
	services = append(services, migration.NewHTTPAPIService(e, cfg.SharedListenAddress, &cfg.HTTPAPIService))
	if cfg.PromMetricsService.Enabled {
		services = append(services, migration.NewPromMetricsService(e, &cfg.PromMetricsService))
	}

	done := registerShutdownHooks(cfg, services...)
	initServices(services...)
	runServices(services...)

	log.Println("service shutdown successfully")
	<-done
}

// service can be initialized, run and be shut down.
type service interface {
	// Init inits the service.
	Init() error
	// Run runs the service.
	Run() error
	// Shutdown shuts down the service.
	Shutdown(context.Context) error
}

// listen for system signals in order to shutdown the services.
func registerShutdownHooks(cfg *migration.Config, services ...service) <-chan struct{} {
	sigs := make(chan os.Signal, 1)
	done := make(chan struct{}, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		log.Printf("got sigint/sigterm. shutting down... (max await before process kill: %v)", cfg.ShutdownMaxAwait)

		ctx, cancelFunc := context.WithTimeout(context.Background(), cfg.ShutdownMaxAwait)
		defer cancelFunc()

		for _, service := range services {
			if err := service.Shutdown(ctx); err != nil {
				log.Panic(err)
			}
		}

		done <- struct{}{}
	}()

	return done
}

func initServices(services ...service) {
	for _, srv := range services {
		if err := srv.Init(); err != nil {
			log.Panic(err)
		}
	}
}

// runs all given services in their own goroutine.
func runServices(services ...service) {
	var wg sync.WaitGroup
	wg.Add(len(services))
	for _, srv := range services {
		go func(srv service) {
			defer wg.Done()
			if err := srv.Run(); err != nil {
				log.Panic(err)
			}
		}(srv)
	}
	wg.Wait()
}
