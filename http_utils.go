package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

func getCurrentHeight(chainID string, address string) (int, error) {
	url := fmt.Sprintf("%s/status", address)
	
	resp, err := http.Get(url)
	if err != nil {
		Logger("ERROR", ModuleHTTP{ChainID: chainID, Operation: "getCurrentHeight", Success: false, Message: err.Error()})
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		Logger("ERROR", ModuleHTTP{ChainID: chainID, Operation: "getCurrentHeight", Success: false, Message: err.Error()})
		log.Fatal(err)
	}

	var currentHeightResponse CurrentHeightResponse
	err = json.Unmarshal(body, &currentHeightResponse)
	if err != nil {
		log.Fatal(err)
	}

	// check chainID matches nodes chainID
	nodeChainID := currentHeightResponse.Result.NodeInfo.Network
	if nodeChainID != chainID {
		Logger("ERROR", ModuleHTTP{ChainID: chainID, Operation: "getCurrentHeight", Success: false, Message: fmt.Sprintf("Chain ID mismatch: %s != %s", chainID, nodeChainID)})
		log.Fatalf(fmt.Sprintf("ERROR: Chain ID mismatch: %s != %s", chainID, nodeChainID))
	}

	str := currentHeightResponse.Result.SyncInfo.LatestBlockHeight
	num, err := strconv.Atoi(str)
	if err != nil {
		Logger("ERROR", ModuleHTTP{ChainID: chainID, Operation: "getCurrentHeight", Success: false, Message: err.Error()})
	}

	return num, nil
}


func checkBlockSignature(ChainID string, host string, address string, height int, delay string) (string, bool, string){
	if delay != "0ms" {
		delayDuration, err := time.ParseDuration(delay)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(delayDuration)
	}
	url := fmt.Sprintf("%s/block?height=%d", host, height)
	
	resp, err := http.Get(url)
	if err != nil {
		Logger("ERROR", ModuleHTTP{ChainID: ChainID, Operation: "checkBlockSignature", Success: false, Message: err.Error()})
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("ERROR: Failed to read response body: %v", err)
	}

	var blockData BlockResult
	err = json.Unmarshal(body, &blockData)
	if err != nil {
		log.Fatalf("ERROR: Failed to unmarshal JSON: %v", err)
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
	
	Logger("INFO", ModuleHTTP{ChainID: ChainID, Operation: "checkBlockSignature", Height: height, SignatureFound: signatureFound})
	return time,signatureFound,signature
}