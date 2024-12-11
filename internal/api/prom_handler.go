package api

import (
	"cometbftsignrate/internal/config_utils"
	"cometbftsignrate/internal/db_utils"
	"cometbftsignrate/internal/logger"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Declare Prometheus metrics
var (
	SignatureNotFoundCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "signature_not_found_count",
			Help: "Number of signature not found events.",
		},
		[]string{"chainID", "address"},
	)

	SigningRatePercentage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "signing_rate_percentage",
			Help: "Percentage of successful signing.",
		},
		[]string{"chainID", "address"},
	)

	SecondsSinceLatestBlockTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "seconds_since_latest_block_timestamp",
			Help: "Seconds since the latest block timestamp.",
		},
		[]string{"chainID"},
	)
	NumberOfRecordsForChain = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "number_of_records_in_db_for_chain",
			Help: "Number of records in DB for chain.",
		},
		[]string{"chainID"},
	)
	SigningWindowSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "signing_window_size",
			Help: "Signing window size defined in config.toml or if not enough data is available, the value is the number of records available in DB.",
		},
		[]string{"chainID"},
	)
	NumberOfProposedBlocks = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "number_of_proposed_blocks",
			Help: "Number of proposed blocks in signing window.",
		},
		[]string{"chainID", "address", "signing_window"},
	)
	NumberOfEmptyProposedBlocks = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "number_of_empty_proposed_blocks",
			Help: "Number of proposed blocks with zero TXs in them during the signing window.",
		},
		[]string{"chainID", "address", "signing_window"},
	)
)

// Initialize and register Prometheus metrics
func InitMetrics() (*prometheus.Registry, error) {
	customRegistry := prometheus.NewRegistry()

    // Register your custom metrics
    customRegistry.MustRegister(SignatureNotFoundCount)
    customRegistry.MustRegister(SigningRatePercentage)
    customRegistry.MustRegister(SecondsSinceLatestBlockTimestamp)
	customRegistry.MustRegister(NumberOfRecordsForChain)
	customRegistry.MustRegister(SigningWindowSize)
	customRegistry.MustRegister(NumberOfProposedBlocks)
	customRegistry.MustRegister(NumberOfEmptyProposedBlocks)

    return customRegistry, nil
}

// Metrics handler to expose the metrics to Prometheus
func MetricsHandler(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}

// Periodically update metrics every 15 seconds
func StartMetricsUpdater( db *sql.DB, chainID string) {
	logger.PostLog("INFO", fmt.Sprintf("Starting metrics updater for %s...", chainID))
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		updateMetrics(db, config_utils.ChainsData)
	}
}

// Collect and update Prometheus metrics
func updateMetrics(db *sql.DB, chains []config_utils.ChainConfig) {

	for _, chain := range chains {
		// Get the data for this chainID
		count, latestBlockTimestamp, err := db_utils.GetAmountOfSignatureNotFound(db, chain.ChainID, chain.SigningWindow)
		if err != nil {
			fmt.Printf("Error fetching data for chain %s: %v\n", chain.ChainID, err)
			continue
		}

		// Parse the latestBlockTimestamp string to time.Time
		latestBlockTime, err := time.Parse(time.RFC3339, latestBlockTimestamp)
		if err != nil {
			fmt.Printf("Error parsing timestamp for chain %s: %v\n", chain.ChainID, err)
			continue
		}
		duration := time.Since(latestBlockTime)
		roundedDuration := int(duration.Seconds())

		// Calculate the signing rate percentage
		signRate := float64(1) - (float64(count) / float64(chain.SigningWindow)) // Example, adjust accordingly

		// Get number of records in DB for this chain
		numRecords, err := db_utils.GetNumberOfRecordsForChain(db, chain.ChainID)
		if err != nil {
			fmt.Printf("Error fetching number of records for chain %s: %v\n", chain.ChainID, err)
		}
		var window int = chain.SigningWindow
		if numRecords < chain.SigningWindow {
			window = numRecords
		}

		// Convert the signing window to a string
		signingWindowStr := fmt.Sprintf("%d", chain.SigningWindow)

		// Check number of proposed blocks in signing window
		proposedBlocks, err := db_utils.GetNumberOfProposedBlocks(db, chain.ChainID, chain.HexAddress, window)
		if err != nil {
			logger.PostLog("ERROR", logger.ModuleDB{ChainID: chain.ChainID, Operation: "GetNumberOfProposedBlocks", Success: false, Message: err.Error()})
		}
		
		// Check number of empty proposed blocks in signing window
		emptyBlocks, err := db_utils.GetNumberOfEmptyProposedBlocks(db, chain.ChainID, chain.HexAddress, window)
		if err != nil {
			logger.PostLog("ERROR", logger.ModuleDB{ChainID: chain.ChainID, Operation: "GetNumberOfEmptyProposedBlocks", Success: false, Message: err.Error()})
		}
		
		// Update Prometheus metrics
		SignatureNotFoundCount.WithLabelValues(chain.ChainID, chain.HexAddress).Set(float64(count))
		SigningRatePercentage.WithLabelValues(chain.ChainID, chain.HexAddress).Set(signRate)
		SecondsSinceLatestBlockTimestamp.WithLabelValues(chain.ChainID).Set(float64(roundedDuration))
		NumberOfRecordsForChain.WithLabelValues(chain.ChainID).Set(float64(numRecords))
		SigningWindowSize.WithLabelValues(chain.ChainID).Set(float64(window))
		NumberOfProposedBlocks.WithLabelValues(chain.ChainID, chain.HexAddress, signingWindowStr).Set(float64(proposedBlocks))
		NumberOfEmptyProposedBlocks.WithLabelValues(chain.ChainID, chain.HexAddress, signingWindowStr).Set(float64(emptyBlocks))
	}
}
