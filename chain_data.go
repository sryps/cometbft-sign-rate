package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

func processChain(chain Chain, db *sql.DB, initialScan int, sleepDuration int) {
	for {
		// Get current height from RPC (also checks if chainID in config file matches the nodes chainID)
		currentHeight, err := getCurrentHeight(chain.ChainID, chain.HostAddress)
		if err != nil {
			log.Fatalf("Error getting current height for: %s - %v\n", chain.ChainID, err)
		}
		logJSONMessageGeneral("INFO", fmt.Sprintf("Current block height for: %s is %d", chain.ChainID, currentHeight))


		// Get last checked height from DB - if no record exists, use current height less initialScan
		lastCheckedHeight, err := GetLastBlockHeight(db, chain.ChainID)
		if err != nil {
			logJSONMessageGeneral("WARN", fmt.Sprintf("Error getting last checked height for: %s - %v", chain.ChainID, err))
		}

		if lastCheckedHeight == 0 {
			logJSONMessageGeneral("WARN", fmt.Sprintf("Error getting last checked height for: %s - using current height less %d", chain.ChainID, initialScan))
			lastCheckedHeight = currentHeight - initialScan
		}
		logJSONMessageGeneral("INFO", fmt.Sprintf("Last checked height for: %s is %d", chain.ChainID, lastCheckedHeight))

		logJSONMessageGeneral("INFO", fmt.Sprintf("Checking for signatures between %d and %d for: %s", lastCheckedHeight, currentHeight, chain.ChainID))


		// Insert data for all blocks between last checked height and current height
		for i := lastCheckedHeight; i < currentHeight; i++ {
			timestamp, sigFound, signature := checkBlockSignature(chain.ChainID, chain.HostAddress, chain.HexAddress, i, chain.RPCdelay)

			err := InsertBlockHeight(db, timestamp, chain.ChainID, chain.HexAddress, i, sigFound, signature)
			if err != nil {
				logJSONMessageGeneral("ERROR", fmt.Sprintf("Error inserting block to DB for: %s - %v", chain.ChainID, err))
			}
		}
		logJSONMessageGeneral("INFO", fmt.Sprintf("Finished processing signatures for %s, sleeping for %d seconds", chain.ChainID, sleepDuration))

		
		// Prune old records if pruning is enabled - delete records older than the signing window
		if chain.PruningEnabled {
			logJSONMessageGeneral("INFO", fmt.Sprintf("Pruning block data for %s older than %d blocks...", chain.ChainID, chain.SigningWindow))
			DeleteOldRecords(db, chain.ChainID, chain.SigningWindow)
		}

		time.Sleep(time.Duration(sleepDuration) * time.Second)
	}
}

