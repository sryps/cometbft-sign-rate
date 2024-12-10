package db_utils

import (
	"cometbftsignrate/internal/logger"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

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
			logger.PostLog("WARN", logger.ModuleDB{ChainID: chainID, Operation: "DeleteOldRecords", Success: true, Message: "No records to prune"})
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

	logger.PostLog("INFO", logger.ModuleDB{ChainID: chainID, Operation: "DeleteOldRecords", Success: true, Message: "Successfully deleted old records"})
	return nil
}