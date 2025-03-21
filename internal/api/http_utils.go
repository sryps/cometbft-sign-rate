package api

import (
	"cometbftsignrate/internal/logger"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type SyncInfo struct {
	LatestBlockHeight string `json:"latest_block_height"`
}

type CurrentHeightResponse struct {
	Result struct {
		SyncInfo SyncInfo `json:"sync_info"`
		NodeInfo struct {
			Network string `json:"network"`
		} `json:"node_info"`
	} `json:"result"`
}

type BlockResult struct {
	Result struct {
		Block struct {
			Data struct {
				Txs []string `json:"txs"`
			} `json:"data"`
			Header struct {
				Time            string `json:"time"`
				ProposerAddress string `json:"proposer_address"`
			} `json:"header"`
			LastCommit struct {
				Signatures []struct {
					ValidatorAddress string `json:"validator_address"`
					Timestamp        string `json:"timestamp"`
					Signature        string `json:"signature"`
				} `json:"signatures"`
			} `json:"last_commit"`
		} `json:"block"`
	} `json:"result"`
}

func GetCurrentHeight(chainID string, address string) (int, error) {
	url := fmt.Sprintf("%s/status", address)

	resp, err := http.Get(url)
	if err != nil {
		logger.PostLog("ERROR", logger.ModuleHTTP{ChainID: chainID, Operation: "getCurrentHeight", Success: false, Message: err.Error()})
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.PostLog("ERROR", logger.ModuleHTTP{ChainID: chainID, Operation: "getCurrentHeight", Success: false, Message: err.Error()})
		os.Exit(1)
	}

	var currentHeightResponse CurrentHeightResponse
	err = json.Unmarshal(body, &currentHeightResponse)
	if err != nil {
		logger.PostLog("ERROR", logger.ModuleHTTP{ChainID: chainID, Operation: "getCurrentHeight", Success: false, Message: err.Error()})
		os.Exit(1)
	}

	// check chainID matches nodes chainID
	nodeChainID := currentHeightResponse.Result.NodeInfo.Network
	if nodeChainID != chainID {
		logger.PostLog("ERROR", logger.ModuleHTTP{ChainID: chainID, Operation: "getCurrentHeight", Success: false, Message: fmt.Sprintf("Chain ID mismatch: %s != %s", chainID, nodeChainID)})
		os.Exit(1)
	}

	// convert string to int
	str := currentHeightResponse.Result.SyncInfo.LatestBlockHeight
	num, err := strconv.Atoi(str)
	if err != nil {
		logger.PostLog("ERROR", logger.ModuleHTTP{ChainID: chainID, Operation: "getCurrentHeight", Success: false, Message: err.Error()})
		os.Exit(1)
	}

	return num, nil
}

func CheckBlockSignature(ChainID string, host string, address string, height int, delay string) (string, bool, string, string, bool, int, bool) {
	if delay != "0ms" {
		delayDuration, err := time.ParseDuration(delay)
		if err != nil {
			logger.PostLog("ERROR", logger.ModuleHTTP{ChainID: ChainID, Operation: "checkBlockSignature", Success: false, Message: err.Error()})
			os.Exit(1)
		}
		time.Sleep(delayDuration)
	}
	url := fmt.Sprintf("%s/block?height=%d", host, height)

	resp, err := http.Get(url)
	if err != nil {
		logger.PostLog("ERROR", logger.ModuleHTTP{ChainID: ChainID, Operation: "checkBlockSignature", Success: false, Message: err.Error()})
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.PostLog("ERROR", logger.ModuleHTTP{ChainID: ChainID, Operation: "checkBlockSignature", Success: false, Message: err.Error()})
		os.Exit(1)
	}

	var blockData BlockResult
	err = json.Unmarshal(body, &blockData)
	if err != nil {
		logger.PostLog("ERROR", logger.ModuleHTTP{ChainID: ChainID, Operation: "checkBlockSignature", Success: false, Message: err.Error()})
		os.Exit(1)
	}

	// Set block timestamp
	time := blockData.Result.Block.Header.Time

	// Check if signature is found
	var signatureFound bool
	var signature string
	var valTimestamp string
	for _, sig := range blockData.Result.Block.LastCommit.Signatures {
		if sig.ValidatorAddress == address {
			signatureFound = true
			valTimestamp = sig.Timestamp
			signature = sig.Signature
			break
		}
	}

	// Check block proposer address
	var proposerMatch bool
	proposerAddress := blockData.Result.Block.Header.ProposerAddress
	if address == proposerAddress {
		proposerMatch = true
	}

	// Check number of TXs in block
	numTXs := len(blockData.Result.Block.Data.Txs)

	var emptyBlock bool
	if numTXs == 0 {
		emptyBlock = true
	}

	logger.PostLog("INFO", logger.ModuleHTTP{ChainID: ChainID, Operation: "checkBlockSignature", Height: height, SignatureFound: signatureFound})
	return time, signatureFound, valTimestamp, signature, proposerMatch, numTXs, emptyBlock
}
