package api

import (
	"cometbftsignrate/internal/db_utils"
	"cometbftsignrate/internal/logger"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func APIHandler(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	// Get parameters from query
	chainID := r.URL.Query().Get("chainID")
	signingWindowStr := r.URL.Query().Get("signingWindow")

	if chainID == "" || signingWindowStr == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	signingWindow, err := strconv.Atoi(signingWindowStr)
	if err != nil {
		http.Error(w, "Invalid number of records", http.StatusBadRequest)
		return
	}

	// Call the getAmountOfSignatureNotFound function
	count, latestBlockTimestamp, err := db_utils.GetAmountOfSignatureNotFound(db, chainID, signingWindow)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse the latestBlockTimestamp string to time.Time
	latestBlockTime, err := time.Parse(time.RFC3339, latestBlockTimestamp)
	if err != nil {
		http.Error(w, "Invalid latest block timestamp format", http.StatusInternalServerError)
		return
	}
	duration := time.Since(latestBlockTime)
	roundedDuration := int(duration.Seconds())

	// Calculate the signing rate percentage
	signRate := float64(1) - (float64(count) / float64(signingWindow))

	// Get number of records in DB for this chain
	numRecords, err := db_utils.GetNumberOfRecordsForChain(db, chainID)
	if err != nil {
		fmt.Printf("Error fetching number of records for chain %s: %v\n", chainID, err)
	}

	response := map[string]interface{}{
		"chainID": chainID,
		"requestedSigningWindow": signingWindow,
		"missedSignatureCount":   count,
		"signingRatePercentage": signRate,
		"latestBlockTimestamp": latestBlockTimestamp,
		"secondsSinceLatestBlockTimestamp": roundedDuration,
		"availableRecords": numRecords,
	}

	// Set response headers and encode response as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	logger.PostLog("INFO", logger.ModuleHTTP{ChainID: chainID, Operation: "API HTTP Request", Success: true})
}