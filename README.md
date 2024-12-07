# CometBFT Sign Rate

## Overview
CometBFT Sign Rate is a tool designed to monitor and report the signing rate of validators in a CometBFT-based blockchain networks. It helps ensure validators are performing their duties correctly and provides insights into network health.

## Features
- Monitor validator signing rates
- Generate reports on validator performance
- Easily alert on low signing rates
- Easy integration with CometBFT networks

## Installation

Prequisites:
- Golang
 ---

To install CometBFT Sign Rate, clone the repository and run `go build`:

```bash
git clone https://github.com/sryps/cometbft-sign-rate.git
cd cometbft-sign-rate
go build
```

## Usage
To start monitoring, run the following command:

```bash
./cometbftsignrate --config "/path/to/config.toml"
```

## Configuration
Configure the tool by editing the `config.toml` file.
A sample config file is in `config` folder.

Multiple chains can be configured.

```toml
[global]
# Number of seconds to wait before checking the signatures again for each chain
rest_period = 15

# Number of past blocks to check for signatures if the network has never been scanned before
initial_scan = 200

# DB file location
db_location = "./cometbft_signatures.db"

# Port to listen for incoming requests
http_port = 8080


[[chains]]
# Chain ID of the CometBFT network
chain_id = "juno-1"

# RPC endpoint of the chain
host = "http://127.0.0.1:26657"

# HEX pubkey of the validator signing key
address = "A1A1A1A1A1A1A1A1A1A1A1A1A1A1A1A1A1A1A1A1"

# delay between RPC calls in case the node cant handle the load
rpc_delay = "100ms"

# signing window for the validator (number of blocks to check for signatures)
signing_window = 5000

# Enable pruning (pruning removes all records older than signing_window) Default: true
pruning = true
```

## Contact
For questions or support, please open an issue on the GitHub repository.
