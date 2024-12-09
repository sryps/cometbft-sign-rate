package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Chain struct {
	ChainID     string
	HostAddress string
	HexAddress  string
	RPCdelay    string
	SigningWindow int
	PruningEnabled bool
}

type App struct {
	DB *sql.DB
}

var Chains []ChainConfig

func main() {
	// Remove default timestamp from logs
	log.SetFlags(0)

	logJSONMessageGeneral("INFO", "Starting CometBFT signatures service...")

	// Define a cli flag for the config file location
	configFileLocation := flag.String("config", "./config.toml", "Path to the config file")
	flag.Parse()

	// Parse config file for chains
	NewChainConfig()
	config, err := parseConfig(*configFileLocation)
	if err != nil {
		log.Fatalf("Error parsing config file: %v\n", err)
	}

	Chains = append(Chains, config.Chains...)

	// Initialize the SQLite DB
	db, err := initDB(config.GlobalConfig.DbLocation)
	if err != nil {
		log.Fatalf("Error initializing DB: %v\n", err)
		return
	}
	defer CloseDB(db)

    
	// make a channel to handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// Process each chain in a separate goroutine
	// this is better as it will insure that the are non blocking
	// also allows for graceful shutdown and parralel processing
	// benefits are independent operations not constrained by other process cycle time
	// key constraint will be the SQLite DB as it is a shared resource
	for _, chainConfig := range config.Chains {
		chain := Chain(chainConfig)
		wg.Add(2)

		go func(c Chain) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					StartMetricsUpdater(chain, db)
					time.Sleep(2 * time.Second)
				}
			}
		}(chain)
		go func(c Chain) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					processChain(c, db, config.GlobalConfig.InitialScan, config.GlobalConfig.RestPeriod)
					time.Sleep(time.Duration(config.GlobalConfig.RestPeriod) * time.Second)
				}
			}
		}(chain)
	}

	// Set up the HTTP server
	app := &App{DB: db}
	customRegistry, err := InitMetrics()
	if err != nil {
		log.Fatalf("Error initializing metrics: %v\n", err)
	}

	// create a mux/router for handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/signrate", app.amountOfSignatureNotFoundHandler)
	// add prom metrics endpoint - dont need the wrapper around MetricsHandler
	mux.Handle("/metrics", promhttp.HandlerFor(customRegistry, promhttp.HandlerOpts{}))

	srv := &http.Server{
		Addr:         ":" + strconv.Itoa(config.GlobalConfig.HttpPort),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start the server
	// port := ":" + strconv.Itoa(config.GlobalConfig.HttpPort)
	// logJSONMessageGeneral("INFO", "HTTP Server is running on "+port)
	// log.Fatal(http.ListenAndServe(port, nil))
	// Start the HTTP server in a separate goroutine - to alllow for graceful shutdown

	go func() {
		logJSONMessageGeneral("INFO", "HTTP Server is running on "+":"+srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			logJSONMessageGeneral("ERROR", fmt.Sprintf("HTTP server shutdown error: %v", err))
		}
	}()

	<-stop

	logJSONMessageGeneral("INFO", "Shutting down HTTP server... starting graceful shutdown")

	// shutdown the server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != http.ErrServerClosed {
		logJSONMessageGeneral("ERROR", fmt.Sprintf("HTTP server error: %v", err))
	}

	// trigger a cancel on the context to stop the goroutines
	cancel()

	// wait for the goroutines to finish
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logJSONMessageGeneral("INFO", "Graceful shutdown completed")
	case <-time.After(10 * time.Second):
		logJSONMessageGeneral("WARN", "Graceful shutdown timed out")
	}

	// Close the SQLite DB
	CloseDB(db)
	logJSONMessageGeneral("INFO", "Shutdown complete")
}
