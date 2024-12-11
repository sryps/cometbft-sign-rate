# CometBFT Sign Rate

## Overview
CometBFT Sign Rate is a tool designed to monitor and report the signing rate of validators in a CometBFT-based blockchain networks. It helps ensure validators are performing their duties correctly and provides insights into network health.

## Features
- Monitor validator signing rates
- Generate reports on validator performance
- Easily alert on low signing rates
- Easy integration with CometBFT networks
- Persistent data storage with sqlite DB
- Optional pruning to remove unnecessary records and keep DB small

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

This provides an API endpoint and a Prometheus endpoint to collect data from.
See examples below.

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

## Monitoring

### Endpoint: `GET /signrate`

**Description:**
This endpoint retrieves the signing rate for a specified blockchain.

**Query Parameters:**
- `chainID` (string): The ID of the blockchain (e.g., `osmosis-1`).
- `signingWindow` (integer): The window of blocks to calculate the signing rate (e.g., `1000`).

**Example Request:**
```
GET http://127.0.0.1:8080/signrate?chainID=osmosis-1&signingWindow=1000
```

**Example Response:**
```json
{
  "availableRecords": 2047,
  "chainID": "osmosis-1",
  "latestBlockTimestamp": "2024-12-07T20:20:16.045366807Z",
  "missedSignatureCount": 11,
  "requestedSigningWindow": 1000,
  "secondsSinceLatestBlockTimestamp": 1456,
  "signingRatePercentage": 0.989
}
```

**Response Fields:**
- `availableRecords` (integer): The total number of records available for the specified chain the DB.
- `chainID` (string): The Chain ID of the blockchain requested.
- `latestBlockTimestamp` (string): The timestamp of the latest block in the specified chain in the DB.
- `missedSignatureCount` (integer): The number of missed signatures within the requested signing window.
- `requestedSigningWindow` (integer): The window of blocks requested for calculating the signing rate.
- `secondsSinceLatestBlockTimestamp` (integer): The number of seconds since the latest block timestamp in the DB - valuable for making sure data is up to date.
- `signingRatePercentage` (float): The percentage of blocks signed within the requested signing window.

### Endpoint: `GET /metrics`

**Description:**
This endpoint provides various metrics about the CometBFT Sign Rate tool and the monitored blockchains.

**Example Request:**
```
GET http://127.0.0.1:8080/metrics
```

**Example Response:**
```text
# HELP number_of_empty_proposed_blocks Number of proposed blocks with zero TXs in them during the signing window.
# TYPE number_of_empty_proposed_blocks gauge
number_of_empty_proposed_blocks{address="942EE4CEC79B9B74F95681A1C7FEC8A6C9C0389C",chainID="juno-1",signing_window="5000"} 21
number_of_empty_proposed_blocks{address="A16E480524D636B2DA2AD18483327C2E10A5E8A0",chainID="osmosis-1",signing_window="5000"} 0
# HELP number_of_proposed_blocks Number of proposed blocks in signing window.
# TYPE number_of_proposed_blocks gauge
number_of_proposed_blocks{address="942EE4CEC79B9B74F95681A1C7FEC8A6C9C0389C",chainID="juno-1",signing_window="5000"} 21
number_of_proposed_blocks{address="A16E480524D636B2DA2AD18483327C2E10A5E8A0",chainID="osmosis-1",signing_window="5000"} 36
# HELP number_of_records_in_db_for_chain Number of records in DB for chain.
# TYPE number_of_records_in_db_for_chain gauge
number_of_records_in_db_for_chain{chainID="juno-1"} 836
number_of_records_in_db_for_chain{chainID="osmosis-1"} 2010
# HELP seconds_since_latest_block_timestamp Seconds since the latest block timestamp.
# TYPE seconds_since_latest_block_timestamp gauge
seconds_since_latest_block_timestamp{chainID="juno-1"} 1358
seconds_since_latest_block_timestamp{chainID="osmosis-1"} 1499
# HELP signature_not_found_count Number of signature not found events.
# TYPE signature_not_found_count gauge
signature_not_found_count{address="942EE4CEC79B9B74F95681A1C7FEC8A6C9C0389C",chainID="juno-1"} 7
signature_not_found_count{address="A16E480524D636B2DA2AD18483327C2E10A5E8A0",chainID="osmosis-1"} 33
# HELP signing_rate_percentage Percentage of successful signing.
# TYPE signing_rate_percentage gauge
signing_rate_percentage{address="942EE4CEC79B9B74F95681A1C7FEC8A6C9C0389C",chainID="juno-1"} 0.9986
signing_rate_percentage{address="A16E480524D636B2DA2AD18483327C2E10A5E8A0",chainID="osmosis-1"} 0.9934
# HELP signing_window_size Signing window size defined in config.toml or if not enough data is available, the value is the number of records available in DB.
# TYPE signing_window_size gauge
signing_window_size{chainID="juno-1"} 836
signing_window_size{chainID="osmosis-1"} 2010
```

**Response Fields:**
- `number_of_records_in_db_for_chain`: The total count of records stored in the database for each specified blockchain.
- `seconds_since_latest_block_timestamp`: The elapsed time in seconds since the latest block was recorded in the database for each blockchain.
- `signature_not_found_count`: The number of instances where a signature was expected but not found for each validator address on the specified blockchain.
- `signing_rate_percentage`: The percentage of blocks successfully signed by each validator address within the specified signing window on the blockchain.
- `signing_window_size`: The size of the signing window as defined in the configuration file, or the number of records available in the database if the configured window size is not met.

## Contact
For questions or support, please open an issue on the GitHub repository.
