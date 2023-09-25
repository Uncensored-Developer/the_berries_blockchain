package database

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type State struct {
	Balances   map[Account]uint
	txnMempool []Txn
	dbFile     *os.File
}

func NewStateFromDisk() (*State, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	genesisFilePath := filepath.Join(cwd, "database", "genesis.json")
	genesis, err := loadGenesis(genesisFilePath)
	if err != nil {
		return nil, err
	}

	balances := make(map[Account]uint)
	for account, balance := range genesis.Balances {
		balances[account] = balance
	}

	txnDbFilePath := filepath.Join(cwd, "database", "txn.db")
	f, err := os.OpenFile(txnDbFilePath, os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(f)
	state := &State{balances, make([]Txn, 0), f}

	//loop over each of the txn line in the txn.db file
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		var txn Txn
		err := json.Unmarshal(scanner.Bytes(), &txn)
		if err != nil {
			return nil, err
		}

		//Rebuild the state (user balances) as a series of events
		if err := state.apply(txn); err != nil {
			return nil, err
		}
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
func (s *State) Add(txn Txn) error {
	if err := s.apply(txn); err != nil {
		return err
	}

	s.txnMempool = append(s.txnMempool, txn)
	return nil
}

func (s *State) Persist() error {
	//Make a copy of  mempool because the s.txnMempool would be modified
	mempool := make([]Txn, len(s.txnMempool))
	copy(mempool, s.txnMempool)

	for _, m := range mempool {
		txnJson, err := json.Marshal(m)
		if err != nil {
			return err
		}

		if _, err = s.dbFile.Write(append(txnJson, '\n')); err != nil {
			return err
		}
		//remove the txn written to file from the mempool
		s.txnMempool = s.txnMempool[1:]
	}

	return nil
}
