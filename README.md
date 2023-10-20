# THE ONE PIECE BERRIES BLOCKCHAIN (OPBB) LEDGER
This is a simplified implementation of the Ethereum blockchain. This project includes the implementation of 
several blockchain concepts like:

- Immutable & Distributed ledger
- Peer-to-peer system
- Digital signatures to sign and validate transactions
- Gas fees to prevent spamming the network with transactions
- Mining transactions
- Proof of work consensus algorithm
- Block syncing to update your local node with the latest blocks from other peers on the network

This mining process is highly simplified to only have a static block difficulty of 3 as opposed to the 
dynamic block difficulty used by the Ethereum blockchain that can change to make the mining that more or less
time based on how fast blocks are added to the network. Therefore, the hash of the entire block content must
start with 3 leading zeros to be valid and saved to the ledger.

A gas fee of 10 and static gas price of 1 is set for token transfers as opposed to the dynamic gas price system used
by Ethereum based on the current network activity (i.e reducing the gas price when the activity is low and increasing
it as the activity grows).

# HOW TO USE THIS REPOSITORY
1. Install Golang 1.20
2. Clone repository 

### Build project
```
go build ./cmd/tbb
```

### Create a wallet account (set of private and public keys)
```
./tbb wallet new-account --data_dir=$HOME/.opbb
```
```
Enter a password: 
Enter a password: 
New account created: 0x7a398732b9E70950DD6FD25fB3058385Cfe4c116
Saved to: /path/to/keystore
```

### Replace genesis account
Go to file `database/genesis.go` and replace the genesis account `0x0418A658C5874D2Fe181145B685d2e73D761865D` in
variable `genesisJson` with your newly created wallet account from the previous step then **REBUILD** the project

### Run OPBB bootstrap node
```
./tbb run --data_dir=<absolute_path_to_where_data_should_be_stored> --ip=<node_ip> --port=<node_port> --bootstrap_account=<created_wallet_address> --bootstrap_ip=<bootstrap_server_ip> --bootstrap_port=<bootstrap_server_port>
```

### Show available commands and flags
```bash
The Berries Blockchain CLI

Usage:
  tbb [flags]
  tbb [command]

Available Commands:
  balances    Interact with balances (list...)
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  run         Launches the berries blockchain node and its HTTP API.
  wallet      Manages blockchain accounts and keys.

Flags:
  -h, --help   help for tbb

Use "tbb [command] --help" for more information about a command.
```

### Available endpoints on running nodes HTTP server
- `/balances/list` To fetched a list of all the accounts and their balances
- `/txn/add` To send a txn to the node, Request body below:
```bash
{
    "from": "0x0418A658C5874D2Fe181145B685d2e73D761865D",
    "to": "0x486512fA9fbaF06568D13826afe7822842b9E685",
    "password": "<wallet_account_password>",
    "gas": 10,
    "gasPrice": 1,
    "value": 10
}
```
- `/blocks/<height_or_hash>` To get the details of a block using either it's height or hash.
- `/mempool/` To fetch a list of transactions in the mempool.

# Tests
Run all tests with verbosity but one at a time, without timeout, to avoid ports collisions:
```
go test -v -p=1 -timeout=0 ./...
```

# TODO:
- Switch node communication from HTTP to gRPC
