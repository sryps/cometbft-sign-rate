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

	"cometbftsignrate/internal/api"
	"cometbftsignrate/internal/chaindata"
	"cometbftsignrate/internal/config_utils"
	"cometbftsignrate/internal/db_utils"
	"cometbftsignrate/internal/logger"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type App struct {
	DB *sql.DB
}

func main() {
	// Remove default timestamp from logs
	log.SetFlags(0)

	logger.PostLog("INFO", "Starting CometBFT signatures service...")

	// Define a cli flag for the config file location
	configFileLocation := flag.String("config", "./config.toml", "Path to the config file")
	flag.Parse()

	// Set default chain config
	config_utils.SetDefaultChainConfig()

	// Parse the config file
	config, err := config_utils.ParseConfig(*configFileLocation)
	if err != nil {
		logger.PostLog("ERROR", fmt.Sprintf("Error parsing config file: %v", err))
		os.Exit(1)
	}

	config_utils.SetChains(config)

	// Initialize the SQLite DB
	db, err := db_utils.InitDB(config.GlobalConfig.DbLocation)
	if err != nil {
		logger.PostLog("ERROR", fmt.Sprintf("Error initializing DB: %v", err))
		os.Exit(1)
		return
	}
	defer db_utils.CloseDB(db)

    
	// make a channel to handle graceful shutdown
	stopGraceful := make(chan os.Signal, 1)
	stopImmediate := make(chan os.Signal, 1)

	// Handle SIGTERM and SIGINT
	signal.Notify(stopGraceful, syscall.SIGTERM)
	signal.Notify(stopImmediate, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// Process each chain in a separate goroutine for parallel processing
	for _, chainConfig := range config.Chains {
		chain := chaindata.Chain(chainConfig)
		wg.Add(2)

		go func(c chaindata.Chain) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					api.StartMetricsUpdater(db, chain.ChainID)
				}
			}
		}(chain)
		go func(c chaindata.Chain) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					chaindata.ProcessChain(c, db, config.GlobalConfig.InitialScan, config.GlobalConfig.RestPeriod)
					time.Sleep(time.Duration(config.GlobalConfig.RestPeriod) * time.Second)
				}
			}
		}(chain)
	}

	// Set up the HTTP server
	customRegistry, err := api.InitMetrics()
	if err != nil {
		logger.PostLog("ERROR", fmt.Sprintf("Error initializing metrics: %v", err))
		os.Exit(1)
		
	}

	// create a mux/router for handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/signrate", func(w http.ResponseWriter, r *http.Request) {
		api.APIHandler(db, w, r)
	})
	// add prom metrics endpoint - dont need the wrapper around MetricsHandler
	mux.Handle("/metrics", promhttp.HandlerFor(customRegistry, promhttp.HandlerOpts{}))

	srv := &http.Server{
		Addr:         ":" + strconv.Itoa(config.GlobalConfig.HttpPort),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start the HTTP server in a separate goroutine - to alllow for graceful shutdown
	go func() {
		logger.PostLog("INFO", "HTTP Server is running on "+":"+srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			logger.PostLog("ERROR", fmt.Sprintf("HTTP server shutdown error: %v", err))
		}
	}()

	select {
	case <-stopGraceful:
		logger.PostLog("INFO", "Initiating graceful shutdown...")
		// Existing graceful shutdown code
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		
		if err := srv.Shutdown(shutdownCtx); err != http.ErrServerClosed {
			logger.PostLog("ERROR", fmt.Sprintf("HTTP server error: %v", err))
		}
		
		cancel() // Cancel context for goroutines
		
		// Wait for goroutines
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()
		
		select {
		case <-done:
			logger.PostLog("INFO", "Graceful shutdown completed")
		case <-time.After(10 * time.Second):
			logger.PostLog("WARN", "Graceful shutdown timed out")
		}
	
	case <-stopImmediate:
		logger.PostLog("INFO", "Immediate shutdown requested")
		// Immediate shutdown - just exit
		cancel()  // Cancel context for goroutines
		db_utils.CloseDB(db)
		os.Exit(1)
	}

	// Close the SQLite DB
	db_utils.CloseDB(db)
	logger.PostLog("INFO", "Shutdown complete")
}
