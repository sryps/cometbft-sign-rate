package config_utils

import (
	"io"
	"os"

	"github.com/BurntSushi/toml"
)

var ChainsData []ChainConfig

type Config struct {
	GlobalConfig GlobalChainConfig `toml:"global"`
	Chains []ChainConfig `toml:"chains"`
}

type ChainConfig struct {
	ChainID    string `toml:"chain_id"`
	HostAddress    string `toml:"host"`
	HexAddress string `toml:"address"`
	RPCdelay string `toml:"rpc_delay"`
	SigningWindow int `toml:"signing_window"`
	PruningEnabled bool `toml:"pruning"`
}

type GlobalChainConfig struct {
	RestPeriod int `toml:"rest_period"`
	InitialScan int `toml:"initial_scan"`
	DbLocation string `toml:"db_location"`
	HttpPort int `toml:"http_port"`
}

func SetDefaultChainConfig() *ChainConfig {
	return &ChainConfig{
		RPCdelay: "0ms",
		PruningEnabled: true,
	}
}

func ParseConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := toml.Unmarshal(bytes, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func SetChains(config *Config) {
	// Set global chain config
	ChainsData = append(ChainsData, config.Chains...)
}
