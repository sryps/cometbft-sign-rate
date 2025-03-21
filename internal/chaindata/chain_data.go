package chaindata

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"cometbftsignrate/internal/api"
	"cometbftsignrate/internal/db_utils"
	"cometbftsignrate/internal/logger"
)

type Chain struct {
	ChainID        string
	HostAddress    string
	HexAddress     string
	RPCdelay       string
	SigningWindow  int
	PruningEnabled bool
}

func ProcessChain(chain Chain, db *sql.DB, initialScan int, sleepDuration int) {
	for {
		// Get current height from RPC (also checks if chainID in config file matches the nodes chainID)
		currentHeight, err := api.GetCurrentHeight(chain.ChainID, chain.HostAddress)
		if err != nil {
			logger.PostLog("ERROR", logger.ModuleHTTP{ChainID: chain.ChainID, Operation: "ProcessChain", Success: false, Message: err.Error()})
			os.Exit(1)
		}
		logger.PostLog("INFO", logger.ModuleHTTP{ChainID: chain.ChainID, Height: currentHeight, Operation: "getCurrentHeight", Success: true})

		// Get last checked height from DB
		// if no record exists, use current height less initialScan
		// if pruning is enabled, and latest record is older than (current_height - signing_window) use the current height less the signing window
		lastCheckedHeight, err := db_utils.GetLastBlockHeight(db, chain.ChainID, currentHeight, chain.SigningWindow, chain.PruningEnabled)
		if err != nil {
			logger.PostLog("WARN", logger.ModuleDB{ChainID: chain.ChainID, Operation: "GetLastBlockHeight", Success: false})
			logger.PostLog("WARN", "Falling back to using current height less initialScan || signing window")
		}

		if lastCheckedHeight == 0 {
			logger.PostLog("WARN", logger.ModuleDB{ChainID: chain.ChainID, Operation: "GetLastBlockHeight", Success: false, Message: fmt.Sprintf("Using current height less %d", initialScan)})
			lastCheckedHeight = currentHeight - initialScan
		}
		logger.PostLog("INFO", fmt.Sprintf("Chain %s will start syncing from height %d", chain.ChainID, lastCheckedHeight))

		// Insert data for all blocks between last checked height and current height
		for i := lastCheckedHeight; i < currentHeight; i++ {
			timestamp, sigFound, valTimestamp, signature, proposerMatch, numTXs, emptyBlock := api.CheckBlockSignature(chain.ChainID, chain.HostAddress, chain.HexAddress, i, chain.RPCdelay)

			err := db_utils.InsertBlockHeight(db, timestamp, chain.ChainID, chain.HexAddress, i, sigFound, valTimestamp, signature, proposerMatch, numTXs, emptyBlock)
			if err != nil {
				logger.PostLog("ERROR", logger.ModuleDB{ChainID: chain.ChainID, Operation: "InsertBlockHeight", Height: i, SignatureFound: sigFound, Success: false, Message: err.Error()})
				os.Exit(1)
			}
		}
		logger.PostLog("INFO", logger.ModuleDB{ChainID: chain.ChainID, Operation: "InsertBlockHeight", Success: true, Message: fmt.Sprintf("Finished processing signatures, sleeping for %d seconds", sleepDuration)})

		// Prune old records if pruning is enabled - delete records older than the signing window
		if chain.PruningEnabled {
			logger.PostLog("INFO", logger.ModulePruner{ChainID: chain.ChainID, Operation: "PruneOldRecords", Height: currentHeight, Message: fmt.Sprintf("Pruning block data older than %d blocks", chain.SigningWindow)})
			db_utils.DeleteOldRecords(db, chain.ChainID, chain.SigningWindow)
		}

		time.Sleep(time.Duration(sleepDuration) * time.Second)
	}
}
