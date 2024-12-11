package db_utils

import (
	"cometbftsignrate/internal/logger"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// initDB initializes the database and creates the table if it doesn't exist.
func InitDB(dbFile string) (*sql.DB, error ) {
	logger.PostLog("INFO", "Initializing database...")

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
		signaturefound INTEGER NOT NULL DEFAULT 0,
		proposermatch INTEGER NOT NULL DEFAULT 0,
		numtxs INTEGER NOT NULL DEFAULT 0,
		emptyblock INTEGER NOT NULL DEFAULT 0
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
		logger.PostLog("ERROR", fmt.Sprintf("Error closing the database: %v", err))
	}
}
