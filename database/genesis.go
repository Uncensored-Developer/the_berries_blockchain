package database

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"os"
)

type Genesis struct {
	Balances map[common.Address]uint `json:"balances"`
	Symbol   string                  `json:"symbol"`
	ForkOIP1 uint64                  `json:"fork_oip_1"`
}

var genesisJson = `
{
  "genesis_time": "2022-04-19T00:00:00.000000000Z",
  "chain_id": "the-one-piece-berries-ledger",
  "symbol": "OPB",
  "balances": {
    "0x0418A658C5874D2Fe181145B685d2e73D761865D": 1000000
  },
  "fork_oip_1": 10
}
`

func loadGenesis(path string) (Genesis, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Genesis{}, err
	}

	var loadedGenesis Genesis
	err = json.Unmarshal(content, &loadedGenesis)
	if err != nil {
		return Genesis{}, err
	}
	return loadedGenesis, nil
}

func writeGenesisToDisk(path string, genesis []byte) error {
	return os.WriteFile(path, genesis, 0644)
}
