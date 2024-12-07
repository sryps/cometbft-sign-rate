package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// initDB initializes the database and creates the table if it doesn't exist.
func initDB(dbFile string) (*sql.DB, error ) {
	logJSONMessageGeneral("INFO", "Initializing database...")

	if dbFile == "" {
		return nil, fmt.Errorf("dbFile is empty")
	}

	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Create or update the table with the new columns
	createTableSQL := `CREATE TABLE IF NOT EXISTS cometbft_signatures (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp TEXT NOT NULL,
		chain_id TEXT NOT NULL,
		address TEXT NOT NULL,
		block_height INTEGER NOT NULL,
		signature TEXT NOT NULL,
		signatureFound INTEGER NOT NULL DEFAULT 0
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create or update table: %v", err)
	}

	return db, nil
}


// CloseDB closes the database connection.
func CloseDB(db *sql.DB) {
	err := db.Close()
	if err != nil {
		logJSONMessageGeneral("ERROR", "Error closing the database...")
	}
}

func InsertBlockHeight(db *sql.DB, timestamp string, chainID string, address string, blockHeight int, signatureFound bool, signature string) error {
	// Check the highest block_height for the given chain_id
	var latestRecordedBlockHeight sql.NullInt64
	querySQL := `SELECT MAX(block_height) FROM cometbft_signatures WHERE chain_id = ?`
	err := db.QueryRow(querySQL, chainID).Scan(&latestRecordedBlockHeight)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to query max block height: %v", err)
	}

	// Check if the block_height already exists
	var exists bool
	checkSQL := `SELECT EXISTS (SELECT 1 FROM cometbft_signatures WHERE block_height = ?)`
	err = db.QueryRow(checkSQL, blockHeight).Scan(&exists)
	if err != nil {
		log.Fatalf("Failed to check existence: %v", err)
	}

	if !exists {
		// Insert the new row
		insertSQL := `INSERT INTO cometbft_signatures (timestamp, chain_id, address, block_height, signature, signatureFound)
		VALUES (?, ?, ?, ?, ?, ?)`
		_, err = db.Exec(insertSQL, timestamp, chainID, address, blockHeight, signature, signatureFound)
		if err != nil {
			logJSONMessageGeneral("ERROR", fmt.Sprintf("ERROR inserted block height %d for chain %s into DB - %v", blockHeight, chainID, err))
			return err
		}
		logJSONMessageGeneral("INFO", fmt.Sprintf("Successfully inserted block height %d for chain %s into DB", blockHeight, chainID))
	} else {
		logJSONMessageGeneral("WARN", fmt.Sprintf("Block height %d for chain %s already exists in DB", blockHeight, chainID))
	}	
	return nil
}



func GetLastBlockHeight(db *sql.DB, chainID string) (int, error) {
	var blockHeight int
	querySQL := `SELECT block_height FROM cometbft_signatures WHERE chain_id = ? ORDER BY block_height DESC LIMIT 1`
	err := db.QueryRow(querySQL, chainID).Scan(&blockHeight)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil // No rows found, return 0 as the block height
		}
		return 0, fmt.Errorf("failed to get latest block height: %v", err)
	}
	return blockHeight, nil
}

func DeleteOldRecords(db *sql.DB, chainID string, recordCount int) error {
	// Step 1: Get the ID of the `recordCount`-th most recent row for the given chain_id
	query := fmt.Sprintf(`
		WITH RankedRows AS (
			SELECT id
			FROM %s
			WHERE chain_id = $1
			ORDER BY %s DESC
			LIMIT $2
		)
		SELECT id FROM RankedRows
		ORDER BY id ASC
		LIMIT 1;
	`, "cometbft_signatures", "id")

	var thresholdID int
	err := db.QueryRow(query, chainID, recordCount).Scan(&thresholdID)
	if err != nil {
		if err == sql.ErrNoRows {
			logJSONMessageGeneral("WARN", fmt.Sprintf("No records to prune for chain_id %s", chainID))
			return nil
		}
		return fmt.Errorf("failed to get threshold ID: %w", err)
	}

	// Step 2: Delete all records for the given chain_id with an ID less than the threshold
	deleteQuery := fmt.Sprintf(`
		DELETE FROM %s
		WHERE chain_id = $1 AND id < $2;
	`, "cometbft_signatures")

	_, err = db.Exec(deleteQuery, chainID, thresholdID)
	if err != nil {
		return fmt.Errorf("failed to delete old records: %w", err)
	}

	logJSONMessageGeneral("INFO", fmt.Sprintf("Successfully deleted old records for chain_id %s", chainID))
	return nil
}

func getAmountOfSignatureNotFound(db *sql.DB, chainID string, numRecords int) (int, string, error) {
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

	// get lastest block_timestamp
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

func getNumberOfRecordsForChain(db *sql.DB, chainID string) (int, error) {
	var count int
	querySQL := `SELECT COUNT(*) FROM cometbft_signatures WHERE chain_id = ?`
	err := db.QueryRow(querySQL, chainID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get number of records for chain_id %s: %v", chainID, err)
	}
	return count, nil
}