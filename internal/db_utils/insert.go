package db_utils

import (
	"cometbftsignrate/internal/logger"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func InsertBlockHeight(db *sql.DB, timestamp string, chainID string, address string, blockHeight int, signatureFound bool, valTimestamp string, signature string, proposerMatch bool, numTXs int, emptyBlock bool) error {
	// Check the highest block_height for the given chain_id
	var latestRecordedBlockHeight sql.NullInt64
	querySQL := `SELECT MAX(block_height) FROM cometbft_signatures WHERE chain_id = ?`
	err := db.QueryRow(querySQL, chainID).Scan(&latestRecordedBlockHeight)
	if err != nil && err != sql.ErrNoRows {
		logger.PostLog("ERROR", logger.ModuleDB{ChainID: chainID, Operation: "latestRecordedBlockHeight", Height: blockHeight, Success: false, Message: err.Error()})
		return fmt.Errorf("failed to query max block height: %v", err)
	}

	// Check if the block_height already exists
	var exists bool
	checkSQL := `SELECT EXISTS (SELECT 1 FROM cometbft_signatures WHERE block_height = ?)`
	err = db.QueryRow(checkSQL, blockHeight).Scan(&exists)
	if err != nil {
		logger.PostLog("WARN", logger.ModuleDB{ChainID: chainID, Operation: "BlockHeightExist", Height: blockHeight, Success: false, Message: err.Error()})
	}

	if !exists {
		// Insert the new row
		insertSQL := `INSERT INTO cometbft_signatures (timestamp, chain_id, address, block_height, validatortimestamp,signature, signaturefound, proposermatch, numtxs, emptyblock)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		_, err = db.Exec(insertSQL, timestamp, chainID, address, blockHeight, valTimestamp, signature, signatureFound, proposerMatch, numTXs, emptyBlock)
		if err != nil {
			logger.PostLog("ERROR", logger.ModuleDB{ChainID: chainID, Operation: "InsertBlock", Height: blockHeight, Success: false, Message: err.Error()})
			os.Exit(1)
		}
		logger.PostLog("INFO", logger.ModuleDB{ChainID: chainID, Operation: "InsertBlock", Height: blockHeight, SignatureFound: signatureFound, Success: true, Message: "Successfully inserted block height into DB"})
	} else {
		logger.PostLog("WARN", logger.ModuleDB{ChainID: chainID, Operation: "InsertBlock", Height: blockHeight, Success: false, Message: "Block height already exists in DB"})
	}
	return nil
}
