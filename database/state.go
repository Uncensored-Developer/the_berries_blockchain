package database

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type State struct {
	Balances        map[Account]uint
	txnMempool      []Txn
	dbFile          *os.File
	latestBlockHash Hash
}

func (s *State) LatestBlockHash() Hash {
	return s.latestBlockHash
}

func NewStateFromDisk(dataDir string) (*State, error) {
	err := initDataDirIfNotExists(dataDir)
	if err != nil {
		return nil, err
	}

	genesis, err := loadGenesis(getGenesisJsonFilePath(dataDir))
	if err != nil {
		return nil, err
	}

	balances := make(map[Account]uint)
	for account, balance := range genesis.Balances {
		balances[account] = balance
	}

	f, err := os.OpenFile(getBlocksDbFilePath(dataDir), os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(f)
	state := &State{balances, make([]Txn, 0), f, Hash{}}

	//loop over each of the txn line in the txn.db file
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		blockFsJson := scanner.Bytes()
		var blockFs BlockFS
		err = json.Unmarshal(blockFsJson, &blockFs)
		if err != nil {
			return nil, err
		}

		err = state.applyBlock(blockFs.Value)
		if err != nil {
			return nil, err
		}

		state.latestBlockHash = blockFs.Key
	}
	return state, nil
}

func (s *State) apply(txn Txn) error {
	if txn.IsReward() {
		s.Balances[txn.To] += txn.Value
		return nil
	}

	if txn.Value > s.Balances[txn.From] {
		return fmt.Errorf("account %s has insufficient balance for %d", txn.From, txn.Value)
	}

	s.Balances[txn.From] -= txn.Value
	s.Balances[txn.To] += txn.Value

	return nil
}

func (s *State) applyBlock(b Block) error {
	for _, txn := range b.Txns {
		if err := s.apply(txn); err != nil {
			return err
		}
	}
	return nil
}

func (s *State) AddTxn(txn Txn) error {
	if err := s.apply(txn); err != nil {
		return err
	}

	s.txnMempool = append(s.txnMempool, txn)
	return nil
}

func (s *State) AddBlock(b Block) error {
	for _, txn := range b.Txns {
		if err := s.AddTxn(txn); err != nil {
			return err
		}
	}
	return nil
}

func (s *State) Persist() (Hash, error) {
	//Create new Block with only the new transactions
	block := NewBlock(
		s.latestBlockHash,
		uint64(time.Now().Unix()),
		s.txnMempool,
	)
	blockHash, err := block.Hash()
	if err != nil {
		return Hash{}, err
	}

	blockFs := BlockFS{blockHash, block}

	//encode into JSON string
	blockFsJson, err := json.Marshal(blockFs)
	if err != nil {
		return Hash{}, err
	}

	fmt.Println("Saving new Block to disk:")
	fmt.Printf("\t%s\n", blockFsJson)

	_, err = s.dbFile.Write(append(blockFsJson, '\n'))
	if err != nil {
		return Hash{}, err
	}

	s.latestBlockHash = blockHash

	//reset the mempool
	s.txnMempool = []Txn{}
	return s.latestBlockHash, nil
}

func (s *State) Close() {
	s.dbFile.Close()
}
