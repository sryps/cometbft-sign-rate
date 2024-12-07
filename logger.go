package main

import (
	"encoding/json"
	"log"
	"time"
)

// LogEntry defines the structure for log entries
type LogEntrySignature struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"log_level"`
	Message struct {
		
		ChainID   string `json:"chain_id"`
		Height    int    `json:"height"`
		Signature bool `json:"signature"`
	} `json:"message"`
}

type LogEntryGeneral struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"log_level"`
	Message   string `json:"message"`
}


func logJSONMessageSignatures(level, chainID string, height int, signature bool) {
	// Initialize the nested struct
	entry := LogEntrySignature{
		Timestamp: time.Now().Format(time.RFC3339),
			Level:     level,
		Message: struct {
			
			ChainID   string `json:"chain_id"`
			Height    int    `json:"height"`
			Signature bool   `json:"signature"`
		}{
			
			ChainID:   chainID,
			Height:    height,
			Signature: signature,
		},
	}

	// Marshal the log entry to JSON
	logData, err := json.Marshal(entry)
	if err != nil {
		log.Fatalf("Error marshalling log entry: %v", err)
	}

	// Output the JSON log
	log.Println(string(logData))
}

func logJSONMessageGeneral(level, message string) {
	entry := LogEntryGeneral{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   message,
	}

	// Marshal the log entry to JSON
	logData, err := json.Marshal(entry)
	if err != nil {
		log.Fatalf("Error marshalling log entry: %v", err)
	}

	// Output the JSON log
	log.Println(string(logData))
}

