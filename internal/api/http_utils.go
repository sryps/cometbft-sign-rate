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
			Header struct {
				Time string `json:"time"`
			} `json:"header"`
			LastCommit struct {
				Signatures []struct {
					ValidatorAddress string `json:"validator_address"`
					Signature string `json:"signature"`
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


func CheckBlockSignature(ChainID string, host string, address string, height int, delay string) (string, bool, string){
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

	time := blockData.Result.Block.Header.Time

	var signatureFound bool
	var signature string
	for _, sig := range blockData.Result.Block.LastCommit.Signatures {
		if sig.ValidatorAddress == address {
			signatureFound = true
			signature = sig.Signature
			break
		}
	}
	
	logger.PostLog("INFO", logger.ModuleHTTP{ChainID: ChainID, Operation: "checkBlockSignature", Height: height, SignatureFound: signatureFound})
	return time,signatureFound,signature
}