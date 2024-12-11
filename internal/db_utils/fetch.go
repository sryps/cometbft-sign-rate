package db_utils

import (
	"cometbftsignrate/internal/logger"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func GetLastBlockHeight(db *sql.DB, chainID string, currentNodeHeight int, signingWindow int, pruningEnabled bool) (int, error) {
	// Get the latest block height for the given chain_id from DB
	var blockHeight int
	querySQL := `SELECT block_height FROM cometbft_signatures WHERE chain_id = ? ORDER BY block_height DESC LIMIT 1`
	err := db.QueryRow(querySQL, chainID).Scan(&blockHeight)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil // No rows found, return 0 as the block height
		}
		return 0, fmt.Errorf("failed to get latest block height: %v", err)
	}

	// If pruning is enabled, check if the difference between the current node height and the last checked height is greater than the signing window
	if pruningEnabled {
		if currentNodeHeight - blockHeight > signingWindow {
			logger.PostLog("WARN", fmt.Sprintf("Last checked height for %s is older than the signing window (%d)", chainID,signingWindow))
			return currentNodeHeight - signingWindow, nil
		}
	}

	return blockHeight, nil
}


func GetAmountOfSignatureNotFound(db *sql.DB, chainID string, numRecords int) (int, string, error) {
	// Check if the chain_id exists in the database
	var exists bool
	querySQL := `
		SELECT EXISTS (
			SELECT 1
			FROM cometbft_signatures
			WHERE chain_id = ?
			LIMIT 1
		);
	`

	err := db.QueryRow(querySQL, chainID).Scan(&exists)
	if err != nil {
		return 0, "", fmt.Errorf("failed to check if chain_id exists")
	}

	// If the chain_id does not exist, return an error
	if !exists {
		return 0, "", fmt.Errorf("chain_id %s not found", chainID)
	}

	// Get the amount of signatures not found
	var count int
	querySQL = `
		SELECT COUNT(*) 
		FROM (
			SELECT *
			FROM cometbft_signatures 
			WHERE chain_id = ? 
			ORDER BY block_height DESC
			LIMIT ?
		) AS latest_signatures
		WHERE signatureFound = 0;
	`
	
	err = db.QueryRow(querySQL, chainID, numRecords).Scan(&count)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get amount of signatures not found: %v", err)
	}

	// get latest block_timestamp
	querySQL = `
		SELECT timestamp
		FROM cometbft_signatures
		WHERE chain_id = ?
		ORDER BY block_height DESC
		LIMIT 1;
	`
	var timestamp string
	err = db.QueryRow(querySQL, chainID).Scan(&timestamp)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get latest timestamp: %v", err)
	}

	return count, timestamp, nil
}

func GetNumberOfRecordsForChain(db *sql.DB, chainID string) (int, error) {
	// Get the number of records for the given chain_id
	var count int
	querySQL := `SELECT COUNT(*) FROM cometbft_signatures WHERE chain_id = ?`
	err := db.QueryRow(querySQL, chainID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get number of records for chain_id %s: %v", chainID, err)
	}
	return count, nil
}

func GetNumberOfProposedBlocks(db *sql.DB, chainID string, address string, window int) (int, error) {
	// Scan the last X rows and count how many have proposermatch = 1
	var count int
	querySQL := `
		SELECT COUNT(*)
		FROM (
			SELECT proposermatch 
			FROM cometbft_signatures
			WHERE chain_id = ? AND address = ?
			ORDER BY block_height DESC
			LIMIT ?
		) AS last_rows
		WHERE proposermatch = 1`
	err := db.QueryRow(querySQL, chainID, address, window).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count proposed blocks for chain_id %s: %v", chainID, err)
	}
	return count, nil
}

func GetNumberOfEmptyProposedBlocks(db *sql.DB, chainID string, address string, window int) (int, error) {
	// Scan the last X rows and count how many have proposermatch = 1 and numtxs = 0
	var count int
	querySQL := `
		SELECT COUNT(*)
		FROM (
			SELECT proposermatch, numtxs
			FROM cometbft_signatures
			WHERE chain_id = ? AND address = ?
			ORDER BY block_height DESC
			LIMIT ?
		) AS last_rows
		WHERE proposermatch = 1 AND numtxs = 0`
	err := db.QueryRow(querySQL, chainID, address, window).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count proposed blocks with no transactions for chain_id %s: %v", chainID, err)
	}
	return count, nil
}
