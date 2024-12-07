package main

import (
	"database/sql"
	"flag"
	"log"
	"net/http"
	"strconv"
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

    
	// Process each chain in a separate goroutine
	for _, chainConfig := range config.Chains {
		chain := Chain(chainConfig)
		go StartMetricsUpdater(chain, db)
		
		go processChain(chain, db, config.GlobalConfig.InitialScan, config.GlobalConfig.RestPeriod)
	}

	// Set up the HTTP server
	app := &App{DB: db}
	InitMetrics()
	http.HandleFunc("/signrate", app.amountOfSignatureNotFoundHandler)

	// Start the server
	port := ":" + strconv.Itoa(config.GlobalConfig.HttpPort)
	logJSONMessageGeneral("INFO", "HTTP Server is running on " + port)
	log.Fatal(http.ListenAndServe(port, nil))
	
	select {}
}
