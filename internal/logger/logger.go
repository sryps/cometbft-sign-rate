package logger

import (
	"encoding/json"
	"log"
	"time"
)

type LogEntry struct {
	Timestamp string    `json:"timestamp"`
	Level     string    `json:"log_level"`
	Message   *Message  `json:"message,omitempty"`
	ModuleDB  *ModuleDB `json:"module_db,omitempty"`
	ModuleHTTP *ModuleHTTP `json:"module_http,omitempty"`
	ModulePruner *ModulePruner `json:"module_pruner,omitempty"`
}

type Message struct {
	Value string `json:"value"`
}

type ModuleDB struct {
	ChainID   string `json:"chain_id"`
	Operation string `json:"operation"`
	Height    int    `json:"height"`
	SignatureFound bool `json:"signature_found"`
	Success   bool   `json:"success,omitempty"`
	Message   string `json:"message,omitempty"`
}

type ModuleHTTP struct {
	ChainID   string `json:"chain_id"`
	Operation string `json:"operation"`
	Height    int    `json:"height,omitempty"`
	SignatureFound bool `json:"signature_found,omitempty"`
	Success   bool   `json:"success,omitempty"`
	Message   string `json:"message,omitempty"`
}

type ModulePruner struct {
	ChainID   string `json:"chain_id"`
	Operation string `json:"operation"`
	Height    int    `json:"height"`
	Message   string `json:"message,omitempty"`
	Success   bool   `json:"success"`
}

func PostLog(logLevel string, payload interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     logLevel,
	}

	// Set the appropriate field based on payload type
	switch v := payload.(type) {
	case string: // Treat as a Message
		entry.Message = &Message{Value: v}
	case ModuleDB: // Treat as a ModuleDB
		entry.ModuleDB = &v
	case ModuleHTTP: // Treat as a ModuleHTTP
		entry.ModuleHTTP = &v
	case ModulePruner: // Treat as a ModulePruner
		entry.ModulePruner = &v
	default:
		log.Fatalf("Unsupported payload type: %T", v)
	}

	// Marshal the log entry to JSON
	logData, err := json.Marshal(entry)
	if err != nil {
		log.Fatalf("Error marshalling log entry: %v", err)
	}

	// Output the JSON log
	log.Println(string(logData))
}

