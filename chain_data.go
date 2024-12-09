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
		Logger("INFO", ModuleHTTP{ChainID: chain.ChainID, Height: currentHeight, Operation: "getCurrentHeight", Success: true}) 


		// Get last checked height from DB
		// if no record exists, use current height less initialScan
		// if pruning is enabled, and latest record is older than (current_height - signing_window) use the current height less the signing window
		lastCheckedHeight, err := GetLastBlockHeight(db, chain.ChainID, currentHeight, chain.SigningWindow, chain.PruningEnabled)
		if err != nil {
			Logger("WARN", ModuleDB{ChainID: chain.ChainID, Operation: "GetLastBlockHeight", Success: false})
			Logger("WARN", "Falling back to using current height less initialScan || signing window")
		}

		if lastCheckedHeight == 0 {
			Logger("WARN", ModuleDB{ChainID: chain.ChainID, Operation: "GetLastBlockHeight", Success: false, Message: fmt.Sprintf("Using current height less %d", initialScan)})
			lastCheckedHeight = currentHeight - initialScan
		}
		Logger("INFO", ModuleDB{ChainID: chain.ChainID, Operation: "GetLastBlockHeight", Success: true, Height: lastCheckedHeight, Message: fmt.Sprintf("Chain %s will start syncing from height %d", chain.ChainID, lastCheckedHeight)})
		

		// Insert data for all blocks between last checked height and current height
		for i := lastCheckedHeight; i < currentHeight; i++ {
			timestamp, sigFound, signature := checkBlockSignature(chain.ChainID, chain.HostAddress, chain.HexAddress, i, chain.RPCdelay)

			err := InsertBlockHeight(db, timestamp, chain.ChainID, chain.HexAddress, i, sigFound, signature)
			if err != nil {
				Logger("ERROR", ModuleDB{ChainID: chain.ChainID, Operation: "InsertBlockHeight", Height: i, SignatureFound: sigFound, Success: false, Message: err.Error()})
			}
		}
		Logger("INFO", ModuleDB{ChainID: chain.ChainID, Operation: "InsertBlockHeight", Success: true, Message: fmt.Sprintf("Finished processing signatures, sleeping for %d seconds", sleepDuration)})

		
		// Prune old records if pruning is enabled - delete records older than the signing window
		if chain.PruningEnabled {
			Logger("INFO", ModulePruner{ChainID: chain.ChainID, Operation: "PruneOldRecords", Height: currentHeight, Message: fmt.Sprintf("Pruning block data older than %d blocks", chain.SigningWindow)})
			DeleteOldRecords(db, chain.ChainID, chain.SigningWindow)
		}

		time.Sleep(time.Duration(sleepDuration) * time.Second)
	}
}

